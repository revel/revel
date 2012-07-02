package rev

import (
	"database/sql"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
)

// Flash represents a cookie that gets overwritten on each request.
// It allows data to be stored across one page at a time.
// This is commonly used to implement success or error messages.
// e.g. the Post/Redirect/Get pattern: http://en.wikipedia.org/wiki/Post/Redirect/Get
type Flash struct {
	Data, Out map[string]string
}

// These provide a unified view of the request params.
// Includes:
// - URL query string
// - Form values
// - File uploads
type Params struct {
	url.Values
	Files    map[string][]*multipart.FileHeader
	tmpFiles []*os.File // Temp files used during the request.
}

// A signed cookie (and thus limited to 4kb in size).
// Restriction: Keys may not have a colon in them.
type Session map[string]string

type Request struct {
	*http.Request
	ContentType string
}

type Response struct {
	Status      int
	ContentType string
	Cookies     []*http.Cookie

	Out http.ResponseWriter
}

type Controller struct {
	Name       string
	Type       *ControllerType
	MethodType *MethodType

	Request  *Request
	Response *Response

	Flash      Flash                  // User cookie, cleared after each request.
	Session    Session                // Session, stored in cookie, signed.
	Params     *Params                // Parameters from URL and form (including multipart).
	Args       map[string]interface{} // Per-request scratch space.
	RenderArgs map[string]interface{} // Args passed to the template.
	Validation *Validation            // Data validation helpers
	Txn        *sql.Tx                // Nil by default, but may be used by the app / plugins
}

func NewController(w http.ResponseWriter, r *http.Request, ct *ControllerType) *Controller {
	revReq := NewRequest(r)
	flash := restoreFlash(r)
	params := ParseParams(revReq)

	return &Controller{
		Name:    ct.Type.Name(),
		Type:    ct,
		Request: revReq,
		Response: &Response{
			Status:      0,
			ContentType: "",
			Out:         w,
		},

		Params:  params,
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

func NewRequest(r *http.Request) *Request {
	return &Request{r, ContentType(r)}
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
// 2. The rev.Controller embedded type is set to the value provided.
// Returns a value representing a pointer to the new app controller.
func initNewAppController(appControllerType reflect.Type, c *Controller) reflect.Value {
	// It might be a multi-level embedding, so we have to create new controllers
	// at every level of the hierarchy.
	// ASSUME: the first field in each type is the way up to rev.Controller.
	appControllerPtr := reflect.New(appControllerType)
	ptr := appControllerPtr
	for {
		var (
			embeddedField     reflect.Value = ptr.Elem().Field(0)
			embeddedFieldType reflect.Type  = embeddedField.Type()
		)

		// Check if it's the controller.
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
	for key, vals := range c.Params.Values {
		c.Flash.Out[key] = vals[0]
	}
}

func (c *Controller) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Response.Out, cookie)
}

// Invoke the given method, save headers/cookies to the response, and apply the
// result.  (e.g. render a template to the response)
func (c *Controller) Invoke(appControllerPtr reflect.Value, method reflect.Value, methodArgs []reflect.Value) {

	// Handle panics.
	defer func() {
		if err := recover(); err != nil {
			handleInvocationPanic(c, err)
		}
	}()

	// Clean up from the request.
	defer func() {
		// Delete temp files.
		if c.Request.MultipartForm != nil {
			err := c.Request.MultipartForm.RemoveAll()
			if err != nil {
				LOG.Println("Error removing temporary files:", err)
			}
		}

		for _, tmpFile := range c.Params.tmpFiles {
			err := os.Remove(tmpFile.Name())
			if err != nil {
				LOG.Println("W: Could not remove upload temp file:", err)
			}
		}
	}()

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
		Name:  "REVEL_FLASH",
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
		Name:  "REVEL_ERRORS",
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
		Name:  "REVEL_SESSION",
		Value: Sign(sessionData) + "-" + sessionData,
		Path:  "/",
	})

	// Apply the result, which generally results in the ResponseWriter getting written.
	result.Apply(c.Request, c.Response)
}

// This function handles a panic in an action invocation.
// It cleans up the stack trace, logs it, and displays an error page.
func handleInvocationPanic(c *Controller, err interface{}) {
	plugins.OnException(c, err)

	// Parse the filename and line from the originating line of app code.
	// /Users/robfig/code/gocode/src/revel/samples/booking/app/controllers/hotels.go:191 (0x44735)
	stack := string(debug.Stack())
	appFrame := strings.Index(stack, BasePath)
	if appFrame == -1 {
		// How embarassing.
		LOG.Println(err, "\n", stack)
		c.Response.Out.WriteHeader(500)
		c.Response.Out.Write([]byte(stack))
		return
	}
	stackElement := stack[appFrame : appFrame+strings.Index(stack[appFrame:], "\n")]
	colonIndex := strings.LastIndex(stackElement, ":")
	filename := stackElement[:colonIndex]
	var line int
	fmt.Sscan(stackElement[colonIndex+1:], &line)

	// Log the trace starting at the app frame.
	LOG.Println(err, "\n", stack[appFrame:])

	// Show an error page.
	description := "Unspecified error"
	if err != nil {
		description = fmt.Sprintln(err)
	}
	ErrorResult{&Error{
		Title:       "Panic",
		Path:        filename[len(BasePath):],
		Line:        line,
		Description: description,
		SourceLines: MustReadLines(filename),
	}}.Apply(c.Request, c.Response)
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
	template, err := MainTemplateLoader.Template(c.Name + "/" + viewName + ".html")
	if err != nil {
		return ErrorResult{err}
	}

	// Determine what method we are in.
	// (e.g. the invoked controller method might have delegated to another method)
	methodType := c.MethodType
	if methodType.Name != viewName {
		methodType = c.Type.Method(viewName)
		if methodType == nil {
			return ErrorResult{fmt.Errorf(
				"No Method %s in Controller %s when loading the view."+
					" (delegating Render is only supported within the same controller)",
				viewName, c.Name)}
		}
	}

	// Get the extra RenderArgs passed in.
	if renderArgNames, ok := methodType.RenderArgNames[line]; ok {
		if len(renderArgNames) == len(extraRenderArgs) {
			for i, extraRenderArg := range extraRenderArgs {
				c.RenderArgs[renderArgNames[i]] = extraRenderArg
			}
		} else {
			LOG.Println(len(renderArgNames), "RenderArg names found for",
				len(extraRenderArgs), "extra RenderArgs")
		}
	} else {
		LOG.Println("No RenderArg names found for Render call on line", line,
			"(Method", methodType, ", ViewName", viewName, ")")
	}

	// Add Validation errors to RenderArgs.
	c.RenderArgs["errors"] = c.Validation.ErrorMap()

	return &RenderTemplateResult{
		Template:   template,
		RenderArgs: c.RenderArgs,
	}
}

// Uses encoding/json.Marshal to return JSON to the client.
func (c *Controller) RenderJson(o interface{}) Result {
	return RenderJsonResult{o}
}

// Uses encoding/xml.Marshal to return XML to the client.
func (c *Controller) RenderXml(o interface{}) Result {
	return RenderXmlResult{o}
}

// Render plaintext in response, printf style.
func (c *Controller) RenderText(text string, objs ...interface{}) Result {
	finalText := text
	if len(objs) > 0 {
		finalText = fmt.Sprintf(text, objs)
	}
	return &RenderTextResult{finalText}
}

// Render a "todo" indicating that the action isn't done yet.
func (c *Controller) Todo() Result {
	c.Response.Status = http.StatusNotImplemented
	return ErrorResult{&Error{
		Title:       "TODO",
		Description: "This action is not implemented",
	}}
}

func (c *Controller) NotFound(msg string) Result {
	c.Response.Status = http.StatusNotFound
	return ErrorResult{&Error{
		Title:       "Not Found",
		Description: msg,
	}}
}

// Return a file, either displayed inline or downloaded as an attachment.
// The name and size are taken from the file info.
func (c *Controller) RenderFile(file *os.File, delivery ContentDisposition) Result {
	var length int64 = -1
	fileInfo, err := file.Stat()
	if err != nil {
		LOG.Println("RenderFile error:", err)
	}
	if fileInfo != nil {
		length = fileInfo.Size()
	}
	return &BinaryResult{
		Reader:   file,
		Name:     filepath.Base(file.Name()),
		Length:   length,
		Delivery: delivery,
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
	if cookie, err := req.Cookie("REVEL_FLASH"); err == nil {
		ParseKeyValueCookie(cookie.Value, func(key, val string) {
			flash.Data[key] = val
		})
	}
	return flash
}

// Restore Validation.Errors from a request.
func restoreValidationErrors(req *http.Request) []*ValidationError {
	errors := make([]*ValidationError, 0, 5)
	if cookie, err := req.Cookie("REVEL_ERRORS"); err == nil {
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
	cookie, err := req.Cookie("REVEL_SESSION")
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
	if len(args) == 0 {
		f.Out["error"] = msg
	} else {
		f.Out["error"] = fmt.Sprintf(msg, args...)
	}
}

func (f Flash) Success(msg string, args ...interface{}) {
	if len(args) == 0 {
		f.Out["success"] = msg
	} else {
		f.Out["success"] = fmt.Sprintf(msg, args...)
	}
}

func ParseParams(req *Request) *Params {
	var files map[string][]*multipart.FileHeader

	// Always want the url parameters.
	values := req.URL.Query()

	// Parse the body depending on the content type.
	switch req.ContentType {
	case "application/x-www-form-urlencoded":
		// Typical form.
		if err := req.ParseForm(); err != nil {
			LOG.Println("Error parsing request body:", err)
		} else {
			for key, vals := range req.Form {
				for _, val := range vals {
					values.Add(key, val)
				}
			}
		}

	case "multipart/form-data":
		// Multipart form.
		// TODO: Extract the multipart form param so app can set it.
		if err := req.ParseMultipartForm(32 << 20 /* 32 MB */); err != nil {
			LOG.Println("Error parsing request body:", err)
		} else {
			for key, vals := range req.MultipartForm.Value {
				for _, val := range vals {
					values.Add(key, val)
				}
			}
			files = req.MultipartForm.File
		}
	}

	return &Params{Values: values, Files: files}
}

func (p *Params) Bind(name string, typ reflect.Type) reflect.Value {
	return Bind(p, name, typ)
}

// Get the content type.
// e.g. From "multipart/form-data; boundary=--" to "multipart/form-data"
// If none is specified, returns "text/html" by default.
func ContentType(req *http.Request) string {
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		return "text/html"
	}
	return strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
}

// Write the header (for now, just the status code).
// The status may be set directly by the application (c.Response.Status = 501).
// if it isn't, then fall back to the provided status code.
func (resp *Response) WriteHeader(defaultStatusCode int, defaultContentType string) {
	if resp.Status == 0 {
		resp.Status = defaultStatusCode
	}
	if resp.ContentType == "" {
		resp.ContentType = defaultContentType
	}
	resp.Out.Header().Set("Content-Type", resp.ContentType)
	resp.Out.WriteHeader(resp.Status)
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
	lowerName      string
}

type MethodArg struct {
	Name string
	Type reflect.Type
}

// Searches for a given exported method (case insensitive)
func (ct *ControllerType) Method(name string) *MethodType {
	lowerName := strings.ToLower(name)
	for _, method := range ct.Methods {
		if method.lowerName == lowerName {
			return method
		}
	}
	return nil
}

var controllers = make(map[string]*ControllerType)

// Register a Controller and its Methods with Revel.
func RegisterController(c interface{}, methods []*MethodType) {
	// De-star the controller type
	// (e.g. given TypeOf((*Application)(nil)), want TypeOf(Application))
	var t reflect.Type = reflect.TypeOf(c)
	var elem reflect.Type = t.Elem()

	// De-star all of the method arg types too.
	for _, m := range methods {
		m.lowerName = strings.ToLower(m.Name)
		for _, arg := range m.Args {
			arg.Type = arg.Type.Elem()
		}
	}

	controllers[strings.ToLower(elem.Name())] = &ControllerType{Type: elem, Methods: methods}
	LOG.Printf("Registered controller: %s", elem.Name())
}

func LookupControllerType(name string) *ControllerType {
	return controllers[strings.ToLower(name)]
}
