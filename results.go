package revel

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"code.google.com/p/go.net/websocket"
)

type Result interface {
	Apply(req *Request, resp *Response)
}

// This result handles all kinds of error codes (500, 404, ..).
// It renders the relevant error page (errors/CODE.format, e.g. errors/500.json).
// If RunMode is "dev", this results in a friendly error page.
type ErrorResult struct {
	RenderArgs map[string]interface{}
	Error      error
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
			r.Error, err)}.Apply(req, resp)
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
	switch e := r.Error.(type) {
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

	if r.RenderArgs == nil {
		r.RenderArgs = make(map[string]interface{})
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

	// need to check if we are on a websocket here
	// net/http panics if we write to a hijacked connection
	if req.Method != "WS" {
		resp.WriteHeader(status, contentType)
		b.WriteTo(resp.Out)
	} else {
		websocket.Message.Send(req.Websocket, fmt.Sprint(revelError))
	}

}

type PlaintextErrorResult struct {
	Error error
}

// This method is used when the template loader or error template is not available.
func (r PlaintextErrorResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusInternalServerError, "text/plain; charset=utf-8")
	resp.Out.Write([]byte(r.Error.Error()))
}

// Action methods return this result to request a template be rendered.
type RenderTemplateResult struct {
	Template   Template
	RenderArgs map[string]interface{}
}

func (r *RenderTemplateResult) Apply(req *Request, resp *Response) {
	// Handle panics when rendering templates.
	defer func() {
		if err := recover(); err != nil {
			ERROR.Println(err)
			PlaintextErrorResult{fmt.Errorf("Template Execution Panic in %s:\n%s",
				r.Template.Name(), err)}.Apply(req, resp)
		}
	}()

	chunked := Config.BoolDefault("results.chunked", false)

	// If it's a HEAD request, throw away the bytes.
	out := io.Writer(resp.Out)
	if req.Method == "HEAD" {
		out = ioutil.Discard
	}

	// In a prod mode, write the status, render, and hope for the best.
	// (In a dev mode, always render to a temporary buffer first to avoid having
	// error pages distorted by HTML already written)
	if chunked && !DevMode {
		resp.WriteHeader(http.StatusOK, "text/html; charset=utf-8")
		r.render(req, resp, out)
		return
	}

	// Render the template into a temporary buffer, to see if there was an error
	// rendering the template.  If not, then copy it into the response buffer.
	// Otherwise, template render errors may result in unpredictable HTML (and
	// would carry a 200 status code)
	var b bytes.Buffer
	r.render(req, resp, &b)
	if !chunked {
		resp.Out.Header().Set("Content-Length", strconv.Itoa(b.Len()))
	}
	resp.WriteHeader(http.StatusOK, "text/html; charset=utf-8")
	b.WriteTo(out)
}

func (r *RenderTemplateResult) render(req *Request, resp *Response, wr io.Writer) {
	err := r.Template.Render(wr, r.RenderArgs)
	if err == nil {
		return
	}

	var templateContent []string
	templateName, line, description := parseTemplateError(err)
	if templateName == "" {
		templateName = r.Template.Name()
		templateContent = r.Template.Content()
	} else {
		if tmpl, err := MainTemplateLoader.Template(templateName); err == nil {
			templateContent = tmpl.Content()
		}
	}
	compileError := &Error{
		Title:       "Template Execution Error",
		Path:        templateName,
		Description: description,
		Line:        line,
		SourceLines: templateContent,
	}
	resp.Status = 500
	ERROR.Printf("Template Execution Error (in %s): %s", templateName, description)
	ErrorResult{r.RenderArgs, compileError}.Apply(req, resp)
}

type RenderHtmlResult struct {
	html string
}

func (r RenderHtmlResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/html; charset=utf-8")
	resp.Out.Write([]byte(r.html))
}

type RenderJsonResult struct {
	obj      interface{}
	callback string
}

func (r RenderJsonResult) Apply(req *Request, resp *Response) {
	var b []byte
	var err error
	if Config.BoolDefault("results.pretty", false) {
		b, err = json.MarshalIndent(r.obj, "", "  ")
	} else {
		b, err = json.Marshal(r.obj)
	}

	if err != nil {
		ErrorResult{Error: err}.Apply(req, resp)
		return
	}

	if r.callback == "" {
		resp.WriteHeader(http.StatusOK, "application/json; charset=utf-8")
		resp.Out.Write(b)
		return
	}

	resp.WriteHeader(http.StatusOK, "application/javascript; charset=utf-8")
	resp.Out.Write([]byte(r.callback + "("))
	resp.Out.Write(b)
	resp.Out.Write([]byte(");"))
}

type RenderXmlResult struct {
	obj interface{}
}

func (r RenderXmlResult) Apply(req *Request, resp *Response) {
	var b []byte
	var err error
	if Config.BoolDefault("results.pretty", false) {
		b, err = xml.MarshalIndent(r.obj, "", "  ")
	} else {
		b, err = xml.Marshal(r.obj)
	}

	if err != nil {
		ErrorResult{Error: err}.Apply(req, resp)
		return
	}

	resp.WriteHeader(http.StatusOK, "application/xml; charset=utf-8")
	resp.Out.Write(b)
}

type RenderTextResult struct {
	text string
}

func (r RenderTextResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/plain; charset=utf-8")
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
	ModTime  time.Time
}

func (r *BinaryResult) Apply(req *Request, resp *Response) {
	disposition := string(r.Delivery)
	if r.Name != "" {
		disposition += fmt.Sprintf(`; filename="%s"`, r.Name)
	}
	resp.Out.Header().Set("Content-Disposition", disposition)

	// If we have a ReadSeeker, delegate to http.ServeContent
	if rs, ok := r.Reader.(io.ReadSeeker); ok {
		// http.ServeContent doesn't know about response.ContentType, so we set the respective header.
		if resp.ContentType != "" {
			resp.Out.Header().Set("Content-Type", resp.ContentType)
		} else {
			contentType := ContentTypeByFilename(r.Name)
			resp.Out.Header().Set("Content-Type", contentType)
		}
		http.ServeContent(resp.Out, req.Request, r.Name, r.ModTime, rs)
	} else {
		// Else, do a simple io.Copy.
		if r.Length != -1 {
			resp.Out.Header().Set("Content-Length", strconv.FormatInt(r.Length, 10))
		}
		resp.WriteHeader(http.StatusOK, ContentTypeByFilename(r.Name))
		io.Copy(resp.Out, r.Reader)
	}

	// Close the Reader if we can
	if v, ok := r.Reader.(io.Closer); ok {
		v.Close()
	}
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
		ERROR.Println("Couldn't resolve redirect:", err.Error())
		ErrorResult{Error: err}.Apply(req, resp)
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
		method := FindMethod(recvType, val)
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
