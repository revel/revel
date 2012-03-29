package play

import (
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
	RenderArgs map[string]interface{} // Args passed to the template.
	Validation *Validation            // Data validation helpers
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

func NewAppController(w http.ResponseWriter, r *http.Request, controllerName, methodName string) (*Controller, reflect.Value) {
	var controllerType *ControllerType = LookupControllerType(controllerName)
	if controllerType == nil {
		LOG.Printf("E: Controller %s not found", controllerName)
		return nil, reflect.ValueOf(nil)
	}

	var (
		// Create an AppController.
		appControllerPtr reflect.Value = reflect.New(controllerType.Type)
		appController    reflect.Value = appControllerPtr.Elem()

		// Create and configure Play Controller
		controller *Controller = NewController(w, r, controllerType)
	)

	// Set the embedded Play Controller field, in the App Controller
	var controllerField reflect.Value = appController.Field(0)
	controllerField.Set(reflect.ValueOf(controller))

	// Set the method being called.
	controller.MethodType = controllerType.Method(methodName)
	if controller.MethodType == nil {
		LOG.Println("E: Failed to find method", methodName, "on Controller",
			controllerName)
		return nil, reflect.ValueOf(nil)
	}

	return controller, appControllerPtr
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
	for _, intc := range getInterceptors(when, appControllerPtr.Type()) {
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
		c.Response.out.Write([]byte(err.Error()))
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

// Redirect to an action within the same Controller.
func (c *Controller) Redirect(val interface{}) Result {
	return &RedirectResult{
		val: val,
	}
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

func (f Flash) Error(msg string) {
	f.Out["error"] = msg
}

func (f Flash) Success(msg string) {
	f.Out["success"] = msg
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
