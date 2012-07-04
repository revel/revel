package rev

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

type Result interface {
	Apply(req *Request, resp *Response)
}

// This result handles all kinds of error codes (500, 404, ..).
// It renders the relevant error page (errors/CODE.format, e.g. errors/500.json).
// If RunMode is "dev", this results in a friendly error page.
type ErrorResult struct {
	RenderArgs map[string]interface{}
	error
}

func (r ErrorResult) Apply(req *Request, resp *Response) {
	format := req.Format
	status := resp.Status
	if status == 0 {
		status = http.StatusInternalServerError
	}

	contentType := ContentTypeByFilename("xxx." + format)
	if contentType == DefaultFileContentType {
		contentType = "text/plain"
	}

	// Get the error template.
	var err error
	templatePath := fmt.Sprintf("errors/%d.%s", status, format)
	tmpl, err := MainTemplateLoader.Template(templatePath)

	// This func shows a plaintext error message, in case the template rendering
	// doesn't work.
	showPlaintext := func(err error) {
		PlaintextErrorResult{fmt.Errorf("Server Error:\n%s\n\n"+
			"Additionally, an error occurred when rendering the error page:\n%s",
			r.error, err)}.Apply(req, resp)
	}

	if tmpl == nil {
		if err == nil {
			err = fmt.Errorf("Couldn't find template %s", templatePath)
		}
		showPlaintext(err)
		return
	}

	// If it's not a revel error, wrap it in one.
	var revelError *Error
	switch e := r.error.(type) {
	case *Error:
		revelError = e
	case error:
		revelError = &Error{
			Title:       "Server Error",
			Description: e.Error(),
		}
	}

	if revelError == nil {
		panic("no error provided")
	}

	r.RenderArgs["RunMode"] = RunMode
	r.RenderArgs["Error"] = revelError
	r.RenderArgs["Router"] = MainRouter

	// Render it.
	var b bytes.Buffer
	err = tmpl.Render(&b, r.RenderArgs)

	// If there was an error, print it in plain text.
	if err != nil {
		showPlaintext(err)
		return
	}

	resp.WriteHeader(status, contentType)
	b.WriteTo(resp.Out)
}

type PlaintextErrorResult struct {
	error
}

// This method is used when the template loader or error template is not available.
func (r PlaintextErrorResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusInternalServerError, "text/plain")
	resp.Out.Write([]byte(r.Error()))
}

// Action methods return this result to request a template be rendered.
type RenderTemplateResult struct {
	Template   Template
	RenderArgs map[string]interface{}
}

func (r *RenderTemplateResult) Apply(req *Request, resp *Response) {
	// Render the template into a temporary buffer, to see if there was an error
	// rendering the template.  If not, then copy it into the response buffer.
	// TODO: It seems a shame to make a copy of everything, but if we don't,
	// template errors result in unpredictable HTML for error pages.
	var b bytes.Buffer
	err := r.Template.Render(&b, r.RenderArgs)
	if err != nil {
		line, description := parseTemplateError(err)
		compileError := &Error{
			Title:       "Template Execution Error",
			Path:        r.Template.Name(),
			Description: description,
			Line:        line,
			SourceLines: r.Template.Content(),
			SourceType:  "template",
		}
		ErrorResult{r.RenderArgs, compileError}.Apply(req, resp)
		return
	}

	resp.WriteHeader(http.StatusOK, "text/html")
	b.WriteTo(resp.Out)
}

type RenderJsonResult struct {
	obj interface{}
}

func (r RenderJsonResult) Apply(req *Request, resp *Response) {
	var b []byte
	var err error
	if RunMode == DEV {
		b, err = json.MarshalIndent(r.obj, "", "  ")
	} else {
		b, err = json.Marshal(r.obj)
	}

	if err != nil {
		ErrorResult{error: err}.Apply(req, resp)
		return
	}

	resp.WriteHeader(http.StatusOK, "application/json")
	resp.Out.Write(b)
}

type RenderXmlResult struct {
	obj interface{}
}

func (r RenderXmlResult) Apply(req *Request, resp *Response) {
	var b []byte
	var err error
	// TODO: Extract indent to app.conf
	if RunMode == DEV {
		b, err = xml.MarshalIndent(r.obj, "", "  ")
	} else {
		b, err = xml.Marshal(r.obj)
	}

	if err != nil {
		ErrorResult{error: err}.Apply(req, resp)
		return
	}

	resp.WriteHeader(http.StatusOK, "application/xml")
	resp.Out.Write(b)
}

type RenderTextResult struct {
	text string
}

func (r RenderTextResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/plain")
	resp.Out.Write([]byte(r.text))
}

type ContentDisposition string

var (
	Attachment ContentDisposition = "attachment"
	Inline     ContentDisposition = "inline"
)

type BinaryResult struct {
	Reader   io.Reader
	Name     string
	Length   int64
	Delivery ContentDisposition
}

func (r *BinaryResult) Apply(req *Request, resp *Response) {
	disposition := string(r.Delivery)
	if r.Name != "" {
		disposition += fmt.Sprintf("; filename=%s;", r.Name)
	}
	resp.Out.Header().Set("Content-Disposition", disposition)

	if r.Length != -1 {
		resp.Out.Header().Set("Content-Length", fmt.Sprintf("%d", r.Length))
	}
	resp.WriteHeader(http.StatusOK, ContentTypeByFilename(r.Name))
	io.Copy(resp.Out, r.Reader)
}

type RedirectToUrlResult struct {
	url string
}

func (r *RedirectToUrlResult) Apply(req *Request, resp *Response) {
	resp.Out.Header().Set("Location", r.url)
	resp.WriteHeader(http.StatusFound, "")
}

type RedirectToActionResult struct {
	val interface{}
}

func (r *RedirectToActionResult) Apply(req *Request, resp *Response) {
	url, err := getRedirectUrl(r.val)
	if err != nil {
		LOG.Println("Couldn't resolve redirect:", err.Error())
		ErrorResult{error: err}.Apply(req, resp)
		return
	}
	resp.Out.Header().Set("Location", url)
	resp.WriteHeader(http.StatusFound, "")
}

func getRedirectUrl(item interface{}) (string, error) {
	// Handle strings
	if url, ok := item.(string); ok {
		return url, nil
	}

	// Handle funcs
	val := reflect.ValueOf(item)
	typ := reflect.TypeOf(item)
	if typ.Kind() == reflect.Func && typ.NumIn() > 0 {
		// Get the Controller Method
		recvType := typ.In(0)
		method := FindMethod(recvType, &val)
		if method == nil {
			return "", errors.New("couldn't find method")
		}

		// Construct the action string (e.g. "Controller.Method")
		if recvType.Kind() == reflect.Ptr {
			recvType = recvType.Elem()
		}
		action := recvType.Name() + "." + method.Name
		actionDef := MainRouter.Reverse(action, make(map[string]string))
		if actionDef == nil {
			return "", errors.New("no route for action " + action)
		}

		return actionDef.String(), nil
	}

	// Out of guesses
	return "", errors.New("didn't recognize type: " + typ.String())
}
