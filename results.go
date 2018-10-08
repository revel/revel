// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

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
	"strings"
	"time"
)

type Result interface {
	Apply(req *Request, resp *Response)
}

// ErrorResult structure used to handles all kinds of error codes (500, 404, ..).
// It renders the relevant error page (errors/CODE.format, e.g. errors/500.json).
// If RunMode is "dev", this results in a friendly error page.
type ErrorResult struct {
	ViewArgs map[string]interface{}
	Error    error
}

var resultsLog = RevelLog.New("section", "results")

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
	lang, _ := r.ViewArgs[CurrentLocaleViewArg].(string)
	// Get the error template.
	var err error
	templatePath := fmt.Sprintf("errors/%d.%s", status, format)
	tmpl, err := MainTemplateLoader.TemplateLang(templatePath, lang)

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
		templateLog.Warn("Got an error rendering template", "error", err, "template", templatePath, "lang", lang)
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

	if r.ViewArgs == nil {
		r.ViewArgs = make(map[string]interface{})
	}
	r.ViewArgs["RunMode"] = RunMode
	r.ViewArgs["DevMode"] = DevMode
	r.ViewArgs["Error"] = revelError
	r.ViewArgs["Router"] = MainRouter

	resultsLog.Info("Rendering error template", "template", templatePath, "error", revelError)

	// Render it.
	var b bytes.Buffer
	err = tmpl.Render(&b, r.ViewArgs)

	// If there was an error, print it in plain text.
	if err != nil {
		templateLog.Warn("Got an error rendering template", "error", err, "template", templatePath, "lang", lang)
		showPlaintext(err)
		return
	}

	// need to check if we are on a websocket here
	// net/http panics if we write to a hijacked connection
	if req.Method == "WS" {
		if err := req.WebSocket.MessageSendJSON(fmt.Sprint(revelError)); err != nil {
			resultsLog.Error("Apply: Send failed", "error", err)
		}
	} else {
		resp.WriteHeader(status, contentType)
		if _, err := b.WriteTo(resp.GetWriter()); err != nil {
			resultsLog.Error("Apply: Response WriteTo failed:", "error", err)
		}
	}

}

type PlaintextErrorResult struct {
	Error error
}

// Apply method is used when the template loader or error template is not available.
func (r PlaintextErrorResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusInternalServerError, "text/plain; charset=utf-8")
	if _, err := resp.GetWriter().Write([]byte(r.Error.Error())); err != nil {
		resultsLog.Error("Apply: Write error:", "error", err)
	}
}

// RenderTemplateResult action methods returns this result to request
// a template be rendered.
type RenderTemplateResult struct {
	Template Template
	ViewArgs map[string]interface{}
}

func (r *RenderTemplateResult) Apply(req *Request, resp *Response) {
	// Handle panics when rendering templates.
	defer func() {
		if err := recover(); err != nil {
			resultsLog.Error("Apply: panic recovery", "error", err)
			PlaintextErrorResult{fmt.Errorf("Template Execution Panic in %s:\n%s",
				r.Template.Name(), err)}.Apply(req, resp)
		}
	}()

	chunked := Config.BoolDefault("results.chunked", false)

	// If it's a HEAD request, throw away the bytes.
	out := io.Writer(resp.GetWriter())
	if req.Method == "HEAD" {
		out = ioutil.Discard
	}

	// In a prod mode, write the status, render, and hope for the best.
	// (In a dev mode, always render to a temporary buffer first to avoid having
	// error pages distorted by HTML already written)
	if chunked && !DevMode {
		resp.WriteHeader(http.StatusOK, "text/html; charset=utf-8")
		if err := r.renderOutput(out); err != nil {
			r.renderError(err, req, resp)
		}
		return
	}

	// Render the template into a temporary buffer, to see if there was an error
	// rendering the template.  If not, then copy it into the response buffer.
	// Otherwise, template render errors may result in unpredictable HTML (and
	// would carry a 200 status code)
	b, err := r.ToBytes()
	if err != nil {
		r.renderError(err, req, resp)
		return
	}

	if !chunked {
		resp.Out.Header().Set("Content-Length", strconv.Itoa(b.Len()))
	}
	resp.WriteHeader(http.StatusOK, "text/html; charset=utf-8")
	if _, err := b.WriteTo(out); err != nil {
		resultsLog.Error("Apply: Response write failed", "error", err)
	}
}

// Return a byte array and or an error object if the template failed to render
func (r *RenderTemplateResult) ToBytes() (b *bytes.Buffer, err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			resultsLog.Error("ApplyBytes: panic recovery", "recover-error", rerr)
			err = fmt.Errorf("Template Execution Panic in %s:\n%s", r.Template.Name(), rerr)
		}
	}()
	b = &bytes.Buffer{}
	if err = r.renderOutput(b); err == nil {
		if Config.BoolDefault("results.trim.html", false) {
			b = r.compressHtml(b)
		}
	}
	return
}

// Output the template to the writer, catch any panics and return as an error
func (r *RenderTemplateResult) renderOutput(wr io.Writer) (err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			resultsLog.Error("ApplyBytes: panic recovery", "recover-error", rerr)
			err = fmt.Errorf("Template Execution Panic in %s:\n%s", r.Template.Name(), rerr)
		}
	}()
	err = r.Template.Render(wr, r.ViewArgs)
	return
}

// Trimming the HTML will do the following:
// * Remove all leading & trailing whitespace on every line
// * Remove all empty lines
// * Attempt to keep formatting inside <pre></pre> tags
//
// This is safe unless white-space: pre; is used in css for formatting.
// Since there is no way to detect that, you will have to keep trimming off in these cases.
func (r *RenderTemplateResult) compressHtml(b *bytes.Buffer) (b2 *bytes.Buffer) {

	// Allocate length of original buffer, so we can write everything without allocating again
	b2.Grow(b.Len())
	insidePre := false
	for {
		text, err := b.ReadString('\n')
		// Convert to lower case for finding <pre> tags.
		tl := strings.ToLower(text)
		if strings.Contains(tl, "<pre>") {
			insidePre = true
		}
		// Trim if not inside a <pre> statement
		if !insidePre {
			// Cut trailing/leading whitespace
			text = strings.Trim(text, " \t\r\n")
			if len(text) > 0 {
				if _, err = b2.WriteString(text); err != nil {
					resultsLog.Error("Apply: ", "error", err)
				}
				if _, err = b2.WriteString("\n"); err != nil {
					resultsLog.Error("Apply: ", "error", err)
				}
			}
		} else {
			if _, err = b2.WriteString(text); err != nil {
				resultsLog.Error("Apply: ", "error", err)
			}
		}
		if strings.Contains(tl, "</pre>") {
			insidePre = false
		}
		// We are finished
		if err != nil {
			break
		}
	}

	return
}

// Render the error in the response
func (r *RenderTemplateResult) renderError(err error, req *Request, resp *Response) {
	compileError, found := err.(*Error)
	if !found {
		var templateContent []string
		templateName, line, description := ParseTemplateError(err)
		if templateName == "" {
			templateLog.Info("Cannot determine template name to render error", "error", err)
			templateName = r.Template.Name()
			templateContent = r.Template.Content()

		} else {
			lang, _ := r.ViewArgs[CurrentLocaleViewArg].(string)
			if tmpl, err := MainTemplateLoader.TemplateLang(templateName, lang); err == nil {
				templateContent = tmpl.Content()
			} else {
				templateLog.Info("Unable to retreive template ", "error", err)
			}
		}
		compileError = &Error{
			Title:       "Template Execution Error",
			Path:        templateName,
			Description: description,
			Line:        line,
			SourceLines: templateContent,
		}
	}
	resp.Status = 500
	resultsLog.Errorf("render: Template Execution Error (in %s): %s", compileError.Path, compileError.Description)
	ErrorResult{r.ViewArgs, compileError}.Apply(req, resp)
}

type RenderHTMLResult struct {
	html string
}

func (r RenderHTMLResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/html; charset=utf-8")
	if _, err := resp.GetWriter().Write([]byte(r.html)); err != nil {
		resultsLog.Error("Apply: Response write failed", "error", err)
	}
}

type RenderJSONResult struct {
	obj      interface{}
	callback string
}

func (r RenderJSONResult) Apply(req *Request, resp *Response) {
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
		if _, err = resp.GetWriter().Write(b); err != nil {
			resultsLog.Error("Apply: Response write failed:", "error", err)
		}
		return
	}

	resp.WriteHeader(http.StatusOK, "application/javascript; charset=utf-8")
	if _, err = resp.GetWriter().Write([]byte(r.callback + "(")); err != nil {
		resultsLog.Error("Apply: Response write failed", "error", err)
	}
	if _, err = resp.GetWriter().Write(b); err != nil {
		resultsLog.Error("Apply: Response write failed", "error", err)
	}
	if _, err = resp.GetWriter().Write([]byte(");")); err != nil {
		resultsLog.Error("Apply: Response write failed", "error", err)
	}
}

type RenderXMLResult struct {
	obj interface{}
}

func (r RenderXMLResult) Apply(req *Request, resp *Response) {
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
	if _, err = resp.GetWriter().Write(b); err != nil {
		resultsLog.Error("Apply: Response write failed", "error", err)
	}
}

type RenderTextResult struct {
	text string
}

func (r RenderTextResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/plain; charset=utf-8")
	if _, err := resp.GetWriter().Write([]byte(r.text)); err != nil {
		resultsLog.Error("Apply: Response write failed", "error", err)
	}
}

type ContentDisposition string

var (
	NoDisposition ContentDisposition = ""
	Attachment    ContentDisposition = "attachment"
	Inline        ContentDisposition = "inline"
)

type BinaryResult struct {
	Reader   io.Reader
	Name     string
	Length   int64
	Delivery ContentDisposition
	ModTime  time.Time
}

func (r *BinaryResult) Apply(req *Request, resp *Response) {
	if r.Delivery != NoDisposition {
		disposition := string(r.Delivery)
		if r.Name != "" {
			disposition += fmt.Sprintf(`; filename="%s"`, r.Name)
		}
		resp.Out.internalHeader.Set("Content-Disposition", disposition)
	}
	if resp.ContentType != "" {
		resp.Out.internalHeader.Set("Content-Type", resp.ContentType)
	} else {
		contentType := ContentTypeByFilename(r.Name)
		resp.Out.internalHeader.Set("Content-Type", contentType)
	}
	if content, ok := r.Reader.(io.ReadSeeker); ok && r.Length < 0 {
		// get the size from the stream
		// go1.6 compatibility change, go1.6 does not define constants io.SeekStart
		//if size, err := content.Seek(0, io.SeekEnd); err == nil {
		//	if _, err = content.Seek(0, io.SeekStart); err == nil {
		if size, err := content.Seek(0, 2); err == nil {
			if _, err = content.Seek(0, 0); err == nil {
				r.Length = size
			}
		}
	}

	// Write stream writes the status code to the header as well
	if ws := resp.GetStreamWriter(); ws != nil {
		if err := ws.WriteStream(r.Name, r.Length, r.ModTime, r.Reader); err != nil {
			resultsLog.Error("Apply: Response write failed", "error", err)
		}
	}

	// Close the Reader if we can
	if v, ok := r.Reader.(io.Closer); ok {
		_ = v.Close()
	}
}

type RedirectToURLResult struct {
	url string
}

func (r *RedirectToURLResult) Apply(req *Request, resp *Response) {
	resp.Out.internalHeader.Set("Location", r.url)
	resp.WriteHeader(http.StatusFound, "")
}

type RedirectToActionResult struct {
	val  interface{}
	args []interface{}
}

func (r *RedirectToActionResult) Apply(req *Request, resp *Response) {
	url, err := getRedirectURL(r.val, r.args)
	if err != nil {
		resultsLog.Error("Apply: Couldn't resolve redirect", "error", err)
		ErrorResult{Error: err}.Apply(req, resp)
		return
	}
	resp.Out.internalHeader.Set("Location", url)
	resp.WriteHeader(http.StatusFound, "")
}

func getRedirectURL(item interface{}, args []interface{}) (string, error) {
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
		module := ModuleFromPath(recvType.PkgPath(), true)
		action := module.Namespace() + recvType.Name() + "." + method.Name
		// Fetch the action path to get the defaults
		pathData, found := splitActionPath(nil, action, true)
		if !found {
			return "", fmt.Errorf("Unable to redirect '%s', expected 'Controller.Action'", action)
		}

		// Build the map for the router to reverse
		// Unbind the arguments.
		argsByName := make(map[string]string)
		// Bind any static args first
		fixedParams := len(pathData.FixedParamsByName)
		methodType := pathData.TypeOfController.Method(pathData.MethodName)

		for i, argValue := range args {
			Unbind(argsByName, methodType.Args[i+fixedParams].Name, argValue)
		}

		actionDef := MainRouter.Reverse(action, argsByName)
		if actionDef == nil {
			return "", errors.New("no route for action " + action)
		}

		return actionDef.String(), nil
	}

	// Out of guesses
	return "", errors.New("didn't recognize type: " + typ.String())
}
