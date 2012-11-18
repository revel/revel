package rev

import (
	"net/http"
	"reflect"
	"strings"
)

type Request struct {
	*http.Request
	ContentType    string
	Format         string // "html", "xml", "json", or "text"
	AcceptLanguage []string
}

type Response struct {
	Status      int
	ContentType string

	Out http.ResponseWriter
}

func NewRequest(r *http.Request) *Request {
	return &Request{
		Request:        r,
		ContentType:    ResolveContentType(r),
		Format:         ResolveFormat(r),
		AcceptLanguage: ResolveAcceptLanguage(r),
	}
}

func NewResponse(w http.ResponseWriter) *Response {
	return &Response{Out: w}
}

var (
	controllerType    = reflect.TypeOf(Controller{})
	controllerPtrType = reflect.TypeOf(&Controller{})
)

func NewAppController(req *Request, resp *Response, controllerName, methodName string) (*Controller, reflect.Value) {
	var appControllerType *ControllerType = LookupControllerType(controllerName)
	if appControllerType == nil {
		INFO.Printf("Controller %s not found: %s", controllerName, req.URL)
		return nil, reflect.ValueOf(nil)
	}

	controller := NewController(req, resp, appControllerType)
	appControllerPtr := initNewAppController(appControllerType.Type, controller)

	// Set the method being called.
	controller.MethodType = appControllerType.Method(methodName)
	if controller.MethodType == nil {
		INFO.Println("Failed to find method", methodName, "on Controller",
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

func RenderError(req *Request, resp *Response, err error) {
	stubController(req, resp).RenderError(err).Apply(req, resp)
}

// This function is useful if there is no relevant Controller available.
// It writes the 404 response immediately.
func NotFound(req *Request, resp *Response, msg string) {
	stubController(req, resp).NotFound(msg).Apply(req, resp)
}

func stubController(req *Request, resp *Response) *Controller {
	return &Controller{
		Response: resp,
		RenderArgs: map[string]interface{}{
			"RunMode": RunMode,
		},
	}
}

// Get the content type.
// e.g. From "multipart/form-data; boundary=--" to "multipart/form-data"
// If none is specified, returns "text/html" by default.
func ResolveContentType(req *http.Request) string {
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		return "text/html"
	}
	return strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
}

func ResolveFormat(req *http.Request) string {
	accept := req.Header.Get("accept")

	switch {
	case accept == "",
		strings.HasPrefix(accept, "*/*"),
		strings.Contains(accept, "application/xhtml"),
		strings.Contains(accept, "text/html"):
		return "html"
	case strings.Contains(accept, "application/xml"),
		strings.Contains(accept, "text/xml"):
		return "xml"
	case strings.Contains(accept, "text/plain"):
		return "txt"
	case strings.Contains(accept, "application/json"),
		strings.Contains(accept, "text/javascript"):
		return "json"
	}

	return "html"
}

// Resolve the Accept-Language header value.
func ResolveAcceptLanguage(req *http.Request) []string {
	header := req.Header.Get("Accept-Language")
	if header == "" {
		return nil
	}

	acceptLanguages := strings.Split(header, ",")
	for i, languageRange := range acceptLanguages {
		acceptLanguages[i] = strings.TrimSpace(languageRange)
	}
	return acceptLanguages
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
	TRACE.Printf("Registered controller: %s", elem.Name())
}

func LookupControllerType(name string) *ControllerType {
	return controllers[strings.ToLower(name)]
}
