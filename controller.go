package rev

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
)

type Controller struct {
	Name          string          // The controller name, e.g. "Application"
	Type          *ControllerType // A description of the controller type.
	MethodType    *MethodType     // A description of the invoked action type.
	AppController interface{}     // The controller that was instantiated.
	Action        string          // The full action name, e.g. "Application.Index"

	Request  *Request
	Response *Response
	Result   Result

	Flash      Flash                  // User cookie, cleared after 1 request.
	Session    Session                // Session, stored in cookie, signed.
	Params     *Params                // Parameters from URL and form (including multipart).
	Args       map[string]interface{} // Per-request scratch space.
	RenderArgs map[string]interface{} // Args passed to the template.
	Validation *Validation            // Data validation helpers
	Txn        *sql.Tx                // Nil by default, but may be used by the app / plugins
}

func NewController(req *Request, resp *Response, ct *ControllerType) *Controller {
	c := &Controller{
		Name:     ct.Type.Name(),
		Type:     ct,
		Request:  req,
		Response: resp,
		Params:   ParseParams(req),
		Args:     map[string]interface{}{},
		RenderArgs: map[string]interface{}{
			"RunMode": RunMode,
		},
	}
	c.RenderArgs["Controller"] = c
	return c
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

		plugins.Finally(c)
	}()

	// Clean up from the request.
	defer func() {
		// Delete temp files.
		if c.Request.MultipartForm != nil {
			err := c.Request.MultipartForm.RemoveAll()
			if err != nil {
				WARN.Println("Error removing temporary files:", err)
			}
		}

		for _, tmpFile := range c.Params.tmpFiles {
			err := os.Remove(tmpFile.Name())
			if err != nil {
				WARN.Println("Could not remove upload temp file:", err)
			}
		}
	}()

	// Run the plugins.
	plugins.BeforeRequest(c)

	if c.Result == nil {
		// Invoke the action.
		var resultValue reflect.Value
		if method.Type().IsVariadic() {
			resultValue = method.CallSlice(methodArgs)[0]
		} else {
			resultValue = method.Call(methodArgs)[0]
		}
		if resultValue.Kind() == reflect.Interface && !resultValue.IsNil() {
			c.Result = resultValue.Interface().(Result)
		}

		plugins.AfterRequest(c)
		if c.Result == nil {
			return
		}
	}

	// Apply the result, which generally results in the ResponseWriter getting written.
	c.Result.Apply(c.Request, c.Response)
}

// This function handles a panic in an action invocation.
// It cleans up the stack trace, logs it, and displays an error page.
func handleInvocationPanic(c *Controller, err interface{}) {
	plugins.OnException(c, err)
	stack := string(debug.Stack())
	ERROR.Println(err, "\n", stack)

	error := NewErrorFromPanic(err)
	if error == nil {
		c.Response.Out.WriteHeader(500)
		c.Response.Out.Write([]byte(stack))
		return
	}

	c.RenderError(error).Apply(c.Request, c.Response)
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
//     func (c Users) ShowUser(id int) rev.Result {
//     	 user := loadUser(id)
//     	 return c.Render(user)
//     }
//
// This action will render views/Users/ShowUser.html, passing in an extra
// key-value "user": (User).
func (c *Controller) Render(extraRenderArgs ...interface{}) Result {
	// Get the calling function name.
	pc, _, line, ok := runtime.Caller(1)
	if !ok {
		ERROR.Println("Failed to get Caller information")
		return nil
	}
	// e.g. sample/app/controllers.(*Application).Index
	var fqViewName string = runtime.FuncForPC(pc).Name()
	var viewName string = fqViewName[strings.LastIndex(fqViewName, ".")+1 : len(fqViewName)]

	// Determine what method we are in.
	// (e.g. the invoked controller method might have delegated to another method)
	methodType := c.MethodType
	if methodType.Name != viewName {
		methodType = c.Type.Method(viewName)
		if methodType == nil {
			return c.RenderError(fmt.Errorf(
				"No Method %s in Controller %s when loading the view."+
					" (delegating Render is only supported within the same controller)",
				viewName, c.Name))
		}
	}

	// Get the extra RenderArgs passed in.
	if renderArgNames, ok := methodType.RenderArgNames[line]; ok {
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
			"(Method", methodType, ", ViewName", viewName, ")")
	}

	return c.RenderTemplate(c.Name + "/" + viewName + ".html")
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

// Return a file, either displayed inline or downloaded as an attachment.
// The name and size are taken from the file info.
func (c *Controller) RenderFile(file *os.File, delivery ContentDisposition) Result {
	var length int64 = -1
	fileInfo, err := file.Stat()
	if err != nil {
		WARN.Println("RenderFile error:", err)
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
