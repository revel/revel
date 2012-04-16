package play

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strings"
)

type Flash struct {
	Data, Out map[string]string
}

type Params url.Values

type Session map[string]string

type Request struct {
	*http.Request
}

type Response struct {
	Status      int
	ContentType string
	Headers     http.Header
	Cookies     []*http.Cookie

	out http.ResponseWriter
}

type Controller struct {
	Name       string
	Type       *ControllerType
	MethodType *MethodType

	Request  *Request
	Response *Response

	Flash      Flash                  // User cookie, cleared after each request.
	Session    Session                // Session, stored in cookie, signed.
	Params     Params                 // URL Query parameters
	Args       map[string]interface{} // Per-request scratch space.
	RenderArgs map[string]interface{} // Args passed to the template.
	Validation *Validation            // Data validation helpers
	Txn        *sql.Tx                // Nil by default, but may be used by the app / plugins
}

func NewController(w http.ResponseWriter, r *http.Request, ct *ControllerType) *Controller {
	flash := restoreFlash(r)
	values := r.URL.Query()

	// Add form values to params.
	if err := r.ParseForm(); err != nil {
		LOG.Println("Error parsing request body:", err)
	} else {
		for key, vals := range r.Form {
			for _, val := range vals {
				values.Add(key, val)
			}
		}
	}

	return &Controller{
		Name:    ct.Type.Name(),
		Type:    ct,
		Request: &Request{r},
		Response: &Response{
			Status:      200,
			ContentType: "",
			Headers:     w.Header(),
			out:         w,
		},

		Params:  Params(values),
		Flash:   flash,
		Session: restoreSession(r),
		RenderArgs: map[string]interface{}{
			"flash": flash.Data,
		},
		Validation: &Validation{
			Errors: restoreValidationErrors(r),
			keep:   false,
		},
	}
}

var (
	controllerType    = reflect.TypeOf(Controller{})
	controllerPtrType = reflect.TypeOf(&Controller{})
)

func NewAppController(w http.ResponseWriter, r *http.Request, controllerName, methodName string) (*Controller, reflect.Value) {
	var appControllerType *ControllerType = LookupControllerType(controllerName)
	if appControllerType == nil {
		LOG.Printf("E: Controller %s not found: %s", controllerName, r.URL)
		return nil, reflect.ValueOf(nil)
	}

	controller := NewController(w, r, appControllerType)
	appControllerPtr := initNewAppController(appControllerType.Type, controller)

	// Set the method being called.
	controller.MethodType = appControllerType.Method(methodName)
	if controller.MethodType == nil {
		LOG.Println("E: Failed to find method", methodName, "on Controller",
			controllerName)
		return nil, reflect.ValueOf(nil)
	}

	return controller, appControllerPtr
}

// This is a helper that initializes (zeros) a new app controller value.
// Generally, everything is set to its zero value, except:
// 1. Embedded controller pointers are newed up.
// 2. The play.Controller embedded type is set to the value provided.
// Returns a value representing a pointer to the new app controller.
func initNewAppController(appControllerType reflect.Type, c *Controller) reflect.Value {
	// It might be a multi-level embedding, so we have to create new controllers
	// at every level of the hierarchy.
	// ASSUME: the first field in each type is the way up to play.Controller.
	appControllerPtr := reflect.New(appControllerType)
	ptr := appControllerPtr
	for {
		var (
			embeddedField     reflect.Value = ptr.Elem().Field(0)
			embeddedFieldType reflect.Type  = embeddedField.Type()
		)

		// Check if it's the Play! controller.
		if embeddedFieldType == controllerType {
			embeddedField.Set(reflect.ValueOf(c).Elem())
			break
		} else if embeddedFieldType == controllerPtrType {
			embeddedField.Set(reflect.ValueOf(c))
			break
		}

		// If the embedded field is a pointer, then instantiate an object and set it.
		// (If it's not a pointer, then it's already initialized)
		if embeddedFieldType.Kind() == reflect.Ptr {
			embeddedField.Set(reflect.New(embeddedFieldType.Elem()))
			ptr = embeddedField
		} else {
			ptr = embeddedField.Addr()
		}
	}
	return appControllerPtr
}

func (c *Controller) FlashParams() {
	for key, vals := range c.Params {
		c.Flash.Out[key] = vals[0]
	}
}

func (c *Controller) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Response.out, cookie)
}

// Invoke the given method, save headers/cookies to the response, and apply the
// result.  (e.g. render a template to the response)
func (c *Controller) Invoke(appControllerPtr reflect.Value, method reflect.Value, methodArgs []reflect.Value) {
	// Run the plugins.
	plugins.BeforeRequest(c)

	// Calculate the Result by running the interceptors and the action.
	resultValue := func() reflect.Value {
		// Call the BEFORE interceptors
		result := c.invokeInterceptors(BEFORE, appControllerPtr)
		if result != nil {
			return reflect.ValueOf(result)
		}

		// Invoke the action.
		resultValue := method.Call(methodArgs)[0]

		// Call the AFTER interceptors
		result = c.invokeInterceptors(AFTER, appControllerPtr)
		if result != nil {
			return reflect.ValueOf(result)
		}
		return resultValue
	}()

	plugins.AfterRequest(c)

	if resultValue.IsNil() {
		return
	}
	result := resultValue.Interface().(Result)

	// Store the flash.
	var flashValue string
	for key, value := range c.Flash.Out {
		flashValue += "\x00" + key + ":" + value + "\x00"
	}
	c.SetCookie(&http.Cookie{
		Name:  "PLAY_FLASH",
		Value: url.QueryEscape(flashValue),
		Path:  "/",
	})

	// Store the Validation errors
	var errorsValue string
	if c.Validation.keep {
		for _, error := range c.Validation.Errors {
			if error.Message != "" {
				errorsValue += "\x00" + error.Key + ":" + error.Message + "\x00"
			}
		}
	}
	c.SetCookie(&http.Cookie{
		Name:  "PLAY_ERRORS",
		Value: url.QueryEscape(errorsValue),
		Path:  "/",
	})

	// Store the session (and sign it).
	var sessionValue string
	for key, value := range c.Session {
		sessionValue += "\x00" + key + ":" + value + "\x00"
	}
	sessionData := url.QueryEscape(sessionValue)
	c.SetCookie(&http.Cookie{
		Name:  "PLAY_SESSION",
		Value: Sign(sessionData) + "-" + sessionData,
		Path:  "/",
	})

	// Apply the result, which generally results in the ResponseWriter getting written.
	result.Apply(c.Request, c.Response)
}

func (c *Controller) invokeInterceptors(when InterceptTime, appControllerPtr reflect.Value) Result {
	var result Result
	for _, intc := range getInterceptors(when, appControllerPtr) {
		resultValue := intc.Invoke(appControllerPtr)
		if !resultValue.IsNil() {
			result = resultValue.Interface().(Result)
		}
		if when == BEFORE && result != nil {
			return result
		}
	}
	return result
}

func (c *Controller) Render(extraRenderArgs ...interface{}) Result {
	// Get the calling function name.
	pc, _, line, ok := runtime.Caller(1)
	if !ok {
		log.Println("Failed to get Caller information")
		return nil
	}
	// e.g. sample/app/controllers.(*Application).Index
	var fqViewName string = runtime.FuncForPC(pc).Name()
	var viewName string = fqViewName[strings.LastIndex(fqViewName, ".")+1 : len(fqViewName)]

	// Get the Template.
	template, err := templateLoader.Template(c.Name + "/" + viewName + ".html")
	if err != nil {
		// TODO: Instead of writing output directly, return an error Result
		if err, ok := err.(*CompileError); ok {
			c.Response.out.Write([]byte(err.Html()))
		} else {
			c.Response.out.Write([]byte(err.Error()))
		}
		return nil
	}

	// Get the extra RenderArgs passed in.
	if renderArgNames, ok := c.MethodType.RenderArgNames[line]; ok {
		if len(renderArgNames) == len(extraRenderArgs) {
			for i, extraRenderArg := range extraRenderArgs {
				c.RenderArgs[renderArgNames[i]] = extraRenderArg
			}
		} else {
			LOG.Println(len(renderArgNames), "RenderArg names found for",
				len(extraRenderArgs), "extra RenderArgs")
		}
	} else {
		LOG.Println("No RenderArg names found for Render call on line", line)
	}

	// Add Validation errors to RenderArgs.
	c.RenderArgs["errors"] = c.Validation.ErrorMap()

	return &RenderTemplateResult{
		Template:   template,
		RenderArgs: c.RenderArgs,
		Response:   c.Response,
	}
}

// Redirect to an action or to a URL.
//   c.Redirect(Controller.Action)
//   c.Redirect("/controller/action")
//   c.Redirect("/controller/%d/action", id)
func (c *Controller) Redirect(val interface{}, args ...interface{}) Result {
	if url, ok := val.(string); ok {
		if len(args) == 0 {
			return &RedirectToUrlResult{url}
		}
		return &RedirectToUrlResult{fmt.Sprintf(url, args...)}
	}
	return &RedirectToActionResult{val}
}

// Restore flash from a request.
func restoreFlash(req *http.Request) Flash {
	flash := Flash{
		Data: make(map[string]string),
		Out:  make(map[string]string),
	}
	if cookie, err := req.Cookie("PLAY_FLASH"); err == nil {
		ParseKeyValueCookie(cookie.Value, func(key, val string) {
			flash.Data[key] = val
		})
	}
	return flash
}

// Restore Validation.Errors from a request.
func restoreValidationErrors(req *http.Request) []*ValidationError {
	errors := make([]*ValidationError, 0, 5)
	if cookie, err := req.Cookie("PLAY_ERRORS"); err == nil {
		ParseKeyValueCookie(cookie.Value, func(key, val string) {
			errors = append(errors, &ValidationError{
				Key:     key,
				Message: val,
			})
		})
	}
	return errors
}

func restoreSession(req *http.Request) Session {
	session := make(map[string]string)
	cookie, err := req.Cookie("PLAY_SESSION")
	if err != nil {
		return Session(session)
	}

	// Separate the data from the signature.
	hyphen := strings.Index(cookie.Value, "-")
	if hyphen == -1 || hyphen >= len(cookie.Value)-1 {
		return Session(session)
	}
	sig, data := cookie.Value[:hyphen], cookie.Value[hyphen+1:]

	// Verify the signature.
	if Sign(data) != sig {
		LOG.Println("Session cookie signature failed")
		return Session(session)
	}

	ParseKeyValueCookie(data, func(key, val string) {
		session[key] = val
	})

	return Session(session)
}

func (f Flash) Error(msg string, args ...interface{}) {
	f.Out["error"] = fmt.Sprintf(msg, args)
}

func (f Flash) Success(msg string, args ...interface{}) {
	f.Out["success"] = fmt.Sprintf(msg, args)
}

func (p Params) Bind(valueType reflect.Type, key string) reflect.Value {
	return BindKey(p, valueType, key)
}

// Internal bookeeping

type ControllerType struct {
	Type    reflect.Type
	Methods []*MethodType
}

type MethodType struct {
	Name           string
	Args           []*MethodArg
	RenderArgNames map[int][]string
}

type MethodArg struct {
	Name string
	Type reflect.Type
}

func (ct *ControllerType) Method(name string) *MethodType {
	for _, method := range ct.Methods {
		if method.Name == name {
			return method
		}
	}
	return nil
}

var controllers = make(map[string]*ControllerType)

func RegisterController(c interface{}, methods []*MethodType) {
	// De-star the controller type
	// (e.g. given TypeOf((*Application)(nil)), want TypeOf(Application))
	var t reflect.Type = reflect.TypeOf(c)
	var elem reflect.Type = t.Elem()

	// De-star all of the method arg types too.
	for _, m := range methods {
		for _, arg := range m.Args {
			arg.Type = arg.Type.Elem()
		}
	}

	controllers[elem.Name()] = &ControllerType{Type: elem, Methods: methods}
	log.Printf("Registered controller: %s", elem.Name())
}

func LookupControllerType(name string) *ControllerType {
	return controllers[name]
}
