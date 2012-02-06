package play

import (
	"net/http"
	"net/url"
	"log"
	"reflect"
	"runtime"
	"strings"
)

type Flash map[string]string

type Request struct {
	*http.Request
	Params url.Values
}

type Response struct {
	Status int
	ContentType string
	Headers http.Header
	Cookies []*http.Cookie

	out http.ResponseWriter
}

type Controller struct {
	Name string
	Type *ControllerType
	MethodType *MethodType

	Request  *Request
	Response *Response

	Flash Flash  // User cookie, cleared after each request.
	Session map[string]string  // Session, stored in cookie.
	Params map[string]string
	RenderArgs map[string]interface{}
}

func NewController(w http.ResponseWriter, r *http.Request, ct *ControllerType) *Controller {
	return &Controller{
		Name: ct.Type.Name(),
		Type: ct,
		Request: &Request{r, r.URL.Query()},
		Response: &Response{
			Status: 200,
			ContentType: "",
			Headers: w.Header(),
			out: w,
		},

		Flash: make(map[string]string),
		Session: make(map[string]string),
		RenderArgs: make(map[string]interface{}),
	}
}

func (c *Controller) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Response.out, cookie)
}

// Invoke the given method, save headers/cookies to the response, and apply the
// result.  (e.g. render a template to the response)
func (c *Controller) Invoke(method reflect.Value, methodArgs []reflect.Value) {
	result := method.Call(methodArgs)[0].Interface().(Result)

	// Store the flash.
	var flashValue string
	for key, value := range c.Flash {
		flashValue += "\x00" + key + ":" + value + "\x00"
	}
	c.SetCookie(&http.Cookie{
		Name: "PLAY_FLASH",
		Value: flashValue,
		Path: "/",
	})

	// Apply the result, which generally results in the ResponseWriter getting written.
	result.Apply(c.Request, c.Response)
}

func (c *Controller) Render(extraRenderArgs ...interface{}) Result {
	// Get the calling function name.
	pc, _, line, ok := runtime.Caller(1)
	if ! ok {
		log.Println("Failed to get Caller information")
		return nil
	}
	// e.g. sample/app/controllers.(*Application).Index
	var fqViewName string = runtime.FuncForPC(pc).Name()
	var viewName string = fqViewName[
		strings.LastIndex(fqViewName, ".") + 1 : len(fqViewName)]

	// Refresh templates.
	err := templateLoader.LoadTemplates()
	if err != nil {
		c.Response.out.Write([]byte(err.Html()))
		return nil
	}

	// Get the Template.
	template, err2 := templateLoader.Template(c.Name + "/" + viewName + ".html")
	if err2 != nil {
		c.Response.out.Write([]byte(err2.Error()))
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

	return &RenderTemplateResult{
		Template: template,
		RenderArgs: c.RenderArgs,
		Response: c.Response,
	}
}

// Redirect to an action within the same Controller.
func (c *Controller) Redirect(val interface{}) Result {
	return &RedirectResult{
		val: val,
	}
}

func (f Flash) Error(msg string) {
	f["error"] = msg
}

func (f Flash) Success(msg string) {
	f["success"] = msg
}

// Internal bookeeping

type ControllerType struct {
	Type reflect.Type
	Methods []*MethodType
}

type MethodType struct {
	Name string
	Args []*MethodArg
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

