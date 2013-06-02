package revel

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type Controller struct {
	Name          string          // The controller name, e.g. "Application"
	Type          *ControllerType // A description of the controller type.
	MethodName    string          // The method name, e.g. "Index"
	MethodType    *MethodType     // A description of the invoked action type.
	AppController interface{}     // The controller that was instantiated.
	Action        string          // The fully qualified action name, e.g. "App.Index"

	Request  *Request
	Response *Response
	Result   Result

	Flash      Flash                  // User cookie, cleared after 1 request.
	Session    Session                // Session, stored in cookie, signed.
	Params     *Params                // Parameters from URL and form (including multipart).
	Args       map[string]interface{} // Per-request scratch space.
	RenderArgs map[string]interface{} // Args passed to the template.
	Validation *Validation            // Data validation helpers
}

func NewController(req *Request, resp *Response) *Controller {
	return &Controller{
		Request:  req,
		Response: resp,
		Params:   new(Params),
		Args:     map[string]interface{}{},
		RenderArgs: map[string]interface{}{
			"RunMode": RunMode,
			"DevMode": DevMode,
		},
	}
}

func (c *Controller) FlashParams() {
	for key, vals := range c.Params.Values {
		c.Flash.Out[key] = vals[0]
	}
}

func (c *Controller) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Response.Out, cookie)
}

func (c *Controller) RenderError(err error) Result {
	return ErrorResult{c.RenderArgs, err}
}

// Render a template corresponding to the calling Controller method.
// Arguments will be added to c.RenderArgs prior to rendering the template.
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
func (c *Controller) Render(extraRenderArgs ...interface{}) Result {
	// Get the calling function name.
	_, _, line, ok := runtime.Caller(1)
	if !ok {
		ERROR.Println("Failed to get Caller information")
	}

	// Get the extra RenderArgs passed in.
	if renderArgNames, ok := c.MethodType.RenderArgNames[line]; ok {
		if len(renderArgNames) == len(extraRenderArgs) {
			for i, extraRenderArg := range extraRenderArgs {
				c.RenderArgs[renderArgNames[i]] = extraRenderArg
			}
		} else {
			ERROR.Println(len(renderArgNames), "RenderArg names found for",
				len(extraRenderArgs), "extra RenderArgs")
		}
	} else {
		ERROR.Println("No RenderArg names found for Render call on line", line,
			"(Method", c.MethodType.Name, ")")
	}

	return c.RenderTemplate(c.Name + "/" + c.MethodType.Name + "." + c.Request.Format)
}

// A less magical way to render a template.
// Renders the given template, using the current RenderArgs.
func (c *Controller) RenderTemplate(templatePath string) Result {

	// Get the Template.
	template, err := MainTemplateLoader.Template(templatePath)
	if err != nil {
		return c.RenderError(err)
	}

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
		finalText = fmt.Sprintf(text, objs...)
	}
	return &RenderTextResult{finalText}
}

// Render a "todo" indicating that the action isn't done yet.
func (c *Controller) Todo() Result {
	c.Response.Status = http.StatusNotImplemented
	return c.RenderError(&Error{
		Title:       "TODO",
		Description: "This action is not implemented",
	})
}

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

// Return a file, either displayed inline or downloaded as an attachment.
// The name and size are taken from the file info.
func (c *Controller) RenderFile(file *os.File, delivery ContentDisposition) Result {
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
	return &BinaryResult{
		Reader:   file,
		Name:     filepath.Base(file.Name()),
		Delivery: delivery,
		Length:   -1, // http.ServeContent gets the length itself
		ModTime:  modtime,
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

// Perform a message lookup for the given message name using the given arguments
// using the current language defined for this controller.
//
// The current language is set by the i18n plugin.
func (c *Controller) Message(message string, args ...interface{}) (value string) {
	return Message(c.Request.Locale, message, args...)
}

// SetAction sets the action that is being invoked in the current request.
// It sets the following properties: Name, Action, Type, MethodType
func (c *Controller) SetAction(controllerName, methodName string) error {
	c.Name, c.MethodName = controllerName, methodName
	c.Action = c.Name + "." + c.MethodName

	// Look up the controller and method types.
	var ok bool
	if c.Type, ok = controllers[strings.ToLower(controllerName)]; !ok {
		return errors.New("revel/controller: failed to find controller " + controllerName)
	}
	if c.MethodType = c.Type.Method(methodName); c.MethodType == nil {
		return errors.New("revel/controller: failed to find action " + methodName)
	}

	// Instantiate the controller.
	c.AppController = initNewAppController(c.Type.Type, c).Interface()

	return nil
}

// This is a helper that initializes (zeros) a new app controller value.
// Generally, everything is set to its zero value, except:
// 1. Embedded controller pointers are newed up.
// 2. The revel.Controller embedded type is set to the value provided.
// Returns a value representing a pointer to the new app controller.
func initNewAppController(appControllerType reflect.Type, c *Controller) reflect.Value {
	// It might be a multi-level embedding, so we have to create new controllers
	// at every level of the hierarchy.  To find the controllers, we follow every
	// anonymous field, using breadth-first search.
	appControllerPtr := reflect.New(appControllerType)
	valueQueue := []reflect.Value{appControllerPtr}
	for len(valueQueue) > 0 {
		// Get the next value and de-reference it if necessary.
		var (
			value    = valueQueue[0]
			elem     = value
			elemType = value.Type()
		)
		if elemType.Kind() == reflect.Ptr {
			elem = value.Elem()
			elemType = elem.Type()
		}
		valueQueue = valueQueue[1:]

		// Look at all the struct fields.
		for i := 0; i < elem.NumField(); i++ {
			// If this is not an anonymous field, skip it.
			structField := elemType.Field(i)
			if !structField.Anonymous {
				continue
			}

			fieldValue := elem.Field(i)
			fieldType := structField.Type

			// If it's a Controller, set it to the new instance.
			if fieldType == controllerPtrType {
				fieldValue.Set(reflect.ValueOf(c))
				continue
			}

			// Else, add it to the valueQueue, after instantiating (if necessary).
			if fieldValue.Kind() == reflect.Ptr {
				fieldValue.Set(reflect.New(fieldType.Elem()))
			}
			valueQueue = append(valueQueue, fieldValue)
		}
	}
	return appControllerPtr
}

// Controller registry and types.

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
