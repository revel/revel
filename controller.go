// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
)

// Controller Revel's controller structure that gets embedded in user defined
// controllers
type Controller struct {
	Name          string          // The controller name, e.g. "Application"
	Type          *ControllerType // A description of the controller type.
	MethodName    string          // The method name, e.g. "Index"
	MethodType    *MethodType     // A description of the invoked action type.
	AppController interface{}     // The controller that was instantiated. extends from revel.Controller
	Action        string          // The fully qualified action name, e.g. "App.Index"
	ClientIP      string          // holds IP address of request came from

	Request  *Request
	Response *Response
	Result   Result

	Flash      Flash                  // User cookie, cleared after 1 request.
	Session    Session                // Session, stored in cookie, signed.
	Params     *Params                // Parameters from URL and form (including multipart).
	Args       map[string]interface{} // Per-request scratch space.
	ViewArgs   map[string]interface{} // Variables passed to the template.
	Validation *Validation            // Data validation helpers
}

// NewController returns new controller instance for Request and Response
func NewControllerEmpty() *Controller {
	return &Controller{}
}

// New controller, creates a new instance wrapping the request and response in it
func NewController(req *Request, resp *Response) *Controller {
    c := NewControllerEmpty()
    c.SetController(req, resp)
	return c
}

// Sets the request and the response for the controller
func (c *Controller) SetController(req *Request, resp *Response) {

	c.Request = req
	c.Response = resp
	c.Params = new(Params)
	c.Args = map[string]interface{}{}
	c.ViewArgs = map[string]interface{}{
		"RunMode": RunMode,
		"DevMode": DevMode,
	}

}
func (c *Controller) Destroy() {
	// When the instantiated controller gets injected
	// It inherits this method, so we need to
	// check to see if the controller is nil before performing
	// any actions
	if c == nil {
		return
	}
	if c.AppController != nil {
		c.resetAppControllerFields()
		// Return this instance to the pool
		appController := c.AppController
		c.AppController = nil
		cachedControllerMap[c.Name].Push(appController)
		c.AppController = nil
	}
	c.Request = nil
	c.Response = nil
	c.Params = nil
	c.Args = nil
	c.ViewArgs = nil
	c.Name = ""
	c.Type = nil
	c.MethodName = ""
	c.MethodType = nil
	c.Action = ""
	c.ClientIP = ""
	c.Result = nil
	c.Flash = Flash{}
	c.Session = Session{}
	c.Params = nil
	c.Validation = nil

}

// FlashParams serializes the contents of Controller.Params to the Flash
// cookie.
func (c *Controller) FlashParams() {
	for key, vals := range c.Params.Values {
		c.Flash.Out[key] = strings.Join(vals, ",")
	}
}

func (c *Controller) SetCookie(cookie *http.Cookie) {
	c.Response.Out.Header().SetCookie(cookie.String())

}

func (c *Controller) RenderError(err error) Result {
	c.setStatusIfNil(http.StatusInternalServerError)

	return ErrorResult{c.ViewArgs, err}
}

func (c *Controller) setStatusIfNil(status int) {
	if c.Response.Status == 0 {
		c.Response.Status = status
	}
}

// Render a template corresponding to the calling Controller method.
// Arguments will be added to c.ViewArgs prior to rendering the template.
// They are keyed on their local identifier.
//
// For example:
//
//     func (c Users) ShowUser(id int) revel.Result {
//     	 user := loadUser(id)
//     	 return c.Render(user)
//     }
//
// This action will render views/Users/ShowUser.html, passing in an extra
// key-value "user": (User).
func (c *Controller) Render(extraViewArgs ...interface{}) Result {
	c.setStatusIfNil(http.StatusOK)

	// Get the calling function name.
	_, _, line, ok := runtime.Caller(1)
	if !ok {
		ERROR.Println("Failed to get Caller information")
	}

	// Get the extra ViewArgs passed in.
	if renderArgNames, ok := c.MethodType.RenderArgNames[line]; ok {
		if len(renderArgNames) == len(extraViewArgs) {
			for i, extraRenderArg := range extraViewArgs {
				c.ViewArgs[renderArgNames[i]] = extraRenderArg
			}
		} else {
			ERROR.Println(len(renderArgNames), "RenderArg names found for",
				len(extraViewArgs), "extra ViewArgs")
		}
	} else {
		ERROR.Println("No RenderArg names found for Render call on line", line,
			"(Action", c.Action, ")")
	}

	return c.RenderTemplate(c.Name + "/" + c.MethodType.Name + "." + c.Request.Format)
}

// RenderTemplate method does less magical way to render a template.
// Renders the given template, using the current ViewArgs.
func (c *Controller) RenderTemplate(templatePath string) Result {
	c.setStatusIfNil(http.StatusOK)

	// Get the Template.
	template, err := MainTemplateLoader.Template(templatePath)
	if err != nil {
		return c.RenderError(err)
	}

	return &RenderTemplateResult{
		Template: template,
		ViewArgs: c.ViewArgs,
	}
}

// RenderJSON uses encoding/json.Marshal to return JSON to the client.
func (c *Controller) RenderJSON(o interface{}) Result {
	c.setStatusIfNil(http.StatusOK)

	return RenderJSONResult{o, ""}
}

// RenderJSONP renders JSONP result using encoding/json.Marshal
func (c *Controller) RenderJSONP(callback string, o interface{}) Result {
	c.setStatusIfNil(http.StatusOK)

	return RenderJSONResult{o, callback}
}

// RenderXML uses encoding/xml.Marshal to return XML to the client.
func (c *Controller) RenderXML(o interface{}) Result {
	c.setStatusIfNil(http.StatusOK)

	return RenderXMLResult{o}
}

// RenderText renders plaintext in response, printf style.
func (c *Controller) RenderText(text string, objs ...interface{}) Result {
	c.setStatusIfNil(http.StatusOK)

	finalText := text
	if len(objs) > 0 {
		finalText = fmt.Sprintf(text, objs...)
	}
	return &RenderTextResult{finalText}
}

// RenderHTML renders html in response
func (c *Controller) RenderHTML(html string) Result {
	c.setStatusIfNil(http.StatusOK)

	return &RenderHTMLResult{html}
}

// Todo returns an HTTP 501 Not Implemented "todo" indicating that the
// action isn't done yet.
func (c *Controller) Todo() Result {
	c.Response.Status = http.StatusNotImplemented
	return c.RenderError(&Error{
		Title:       "TODO",
		Description: "This action is not implemented",
	})
}

// NotFound returns an HTTP 404 Not Found response whose body is the
// formatted string of msg and objs.
func (c *Controller) NotFound(msg string, objs ...interface{}) Result {
	finalText := msg
	if len(objs) > 0 {
		finalText = fmt.Sprintf(msg, objs...)
	}
	c.Response.Status = http.StatusNotFound
	return c.RenderError(&Error{
		Title:       "Not Found",
		Description: finalText,
	})
}

// Forbidden returns an HTTP 403 Forbidden response whose body is the
// formatted string of msg and objs.
func (c *Controller) Forbidden(msg string, objs ...interface{}) Result {
	finalText := msg
	if len(objs) > 0 {
		finalText = fmt.Sprintf(msg, objs...)
	}
	c.Response.Status = http.StatusForbidden
	return c.RenderError(&Error{
		Title:       "Forbidden",
		Description: finalText,
	})
}

// RenderFile returns a file, either displayed inline or downloaded
// as an attachment. The name and size are taken from the file info.
func (c *Controller) RenderFile(file *os.File, delivery ContentDisposition) Result {
	c.setStatusIfNil(http.StatusOK)

	var (
		modtime       = time.Now()
		fileInfo, err = file.Stat()
	)
	if err != nil {
		WARN.Println("RenderFile error:", err)
	}
	if fileInfo != nil {
		modtime = fileInfo.ModTime()
	}
	return c.RenderBinary(file, filepath.Base(file.Name()), delivery, modtime)
}

// RenderBinary is like RenderFile() except that it instead of a file on disk,
// it renders data from memory (which could be a file that has not been written,
// the output from some function, or bytes streamed from somewhere else, as long
// it implements io.Reader).  When called directly on something generated or
// streamed, modtime should mostly likely be time.Now().
func (c *Controller) RenderBinary(memfile io.Reader, filename string, delivery ContentDisposition, modtime time.Time) Result {
	c.setStatusIfNil(http.StatusOK)

	return &BinaryResult{
		Reader:   memfile,
		Name:     filename,
		Delivery: delivery,
		Length:   -1, // http.ServeContent gets the length itself unless memfile is a stream.
		ModTime:  modtime,
	}
}

// Redirect to an action or to a URL.
//   c.Redirect(Controller.Action)
//   c.Redirect("/controller/action")
//   c.Redirect("/controller/%d/action", id)
func (c *Controller) Redirect(val interface{}, args ...interface{}) Result {
	c.setStatusIfNil(http.StatusFound)

	if url, ok := val.(string); ok {
		if len(args) == 0 {
			return &RedirectToURLResult{url}
		}
		return &RedirectToURLResult{fmt.Sprintf(url, args...)}
	}
	return &RedirectToActionResult{val}
}

// This stats returns some interesting stats based on what is cached in memory
// and what is available directly
func (c *Controller) Stats() map[string]interface{} {
    result := CurrentEngine.Stats()
    result["revel-controllers"] = controllerStack.String()
    result["revel-requests"] = requestStack.String()
    result["revel-response"] = responseStack.String()
    for key,appStack := range cachedControllerMap {
        result["app-" + key] = appStack.String()
    }
    return result
}
// Message performs a lookup for the given message name using the given
// arguments using the current language defined for this controller.
//
// The current language is set by the i18n plugin.
func (c *Controller) Message(message string, args ...interface{}) string {
	return MessageFunc(c.Request.Locale, message, args...)
}

// SetAction sets the action that is being invoked in the current request.
// It sets the following properties: Name, Action, Type, MethodType
func (c *Controller) SetAction(controllerName, methodName string) error {

	// Look up the controller and method types.
	var ok bool
	if c.Type, ok = controllers[strings.ToLower(controllerName)]; !ok {
		return errors.New("revel/controller: failed to find controller " + controllerName)
	}
	if c.MethodType = c.Type.Method(methodName); c.MethodType == nil {
		return errors.New("revel/controller: failed to find action " + methodName)
	}

	c.Name, c.MethodName = c.Type.Type.Name(), c.MethodType.Name
	c.Action = c.Name + "." + c.MethodName
	if _, ok := cachedControllerMap[c.Name]; !ok {
		// Create a new stack for this controller
		localType := c.Type.Type
		cachedControllerMap[c.Name] = NewStackLock(cachedControllerStackSize, func() interface{} {
			return reflect.New(localType).Interface()
		})
	}
	// Instantiate the controller.
	c.AppController = cachedControllerMap[c.Name].Pop()
	c.setAppControllerFields()

	return nil
}

// Injects this instance (c) into the AppController instance
func (c *Controller) setAppControllerFields() {
	appController := reflect.ValueOf(c.AppController).Elem()
	cValue := reflect.ValueOf(c)
	for _, index := range c.Type.ControllerIndexes {
		appController.FieldByIndex(index).Set(cValue)
	}
}

// Removes this instance (c) from the AppController instance
func (c *Controller) resetAppControllerFields() {
	appController := reflect.ValueOf(c.AppController).Elem()
	// Zero out controller
	for _, index := range c.Type.ControllerIndexes {
		appController.FieldByIndex(index).Set(reflect.Zero(reflect.TypeOf(c.AppController).Elem().FieldByIndex(index).Type))
	}
}

func findControllers(appControllerType reflect.Type) (indexes [][]int) {
	// It might be a multi-level embedding. To find the controllers, we follow
	// every anonymous field, using breadth-first search.
	type nodeType struct {
		val   reflect.Value
		index []int
	}
	appControllerPtr := reflect.New(appControllerType)
	queue := []nodeType{{appControllerPtr, []int{}}}
	for len(queue) > 0 {
		// Get the next value and de-reference it if necessary.
		var (
			node     = queue[0]
			elem     = node.val
			elemType = elem.Type()
		)
		if elemType.Kind() == reflect.Ptr {
			elem = elem.Elem()
			elemType = elem.Type()
		}
		queue = queue[1:]

		// #944 if the type's Kind is not `Struct` move on,
		// otherwise `elem.NumField()` will panic
		if elemType.Kind() != reflect.Struct {
			continue
		}

		// Look at all the struct fields.
		for i := 0; i < elem.NumField(); i++ {
			// If this is not an anonymous field, skip it.
			structField := elemType.Field(i)
			if !structField.Anonymous {
				continue
			}

			fieldValue := elem.Field(i)
			fieldType := structField.Type

			// If it's a Controller, record the field indexes to get here.
			if fieldType == controllerPtrType {
				indexes = append(indexes, append(node.index, i))
				continue
			}

			queue = append(queue,
				nodeType{fieldValue, append(append([]int{}, node.index...), i)})
		}
	}
	return
}

// Controller registry and types.

type ControllerType struct {
	Type              reflect.Type
	Methods           []*MethodType
	ControllerIndexes [][]int // FieldByIndex to all embedded *Controllers
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

// Method searches for a given exported method (case insensitive)
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

// RegisterController registers a Controller and its Methods with Revel.
func RegisterController(c interface{}, methods []*MethodType) {
	// De-star the controller type
	// (e.g. given TypeOf((*Application)(nil)), want TypeOf(Application))
	t := reflect.TypeOf(c)
	elem := t.Elem()

	// De-star all of the method arg types too.
	for _, m := range methods {
		m.lowerName = strings.ToLower(m.Name)
		for _, arg := range m.Args {
			arg.Type = arg.Type.Elem()
		}
	}

	controllers[strings.ToLower(elem.Name())] = &ControllerType{
		Type:              elem,
		Methods:           methods,
		ControllerIndexes: findControllers(elem),
	}
	TRACE.Printf("Registered controller: %s", elem.Name())
}
