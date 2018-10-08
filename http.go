// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"context"
	"mime/multipart"
	"path/filepath"
)

// Request is Revel's HTTP request object structure
type Request struct {
	In              ServerRequest   // The server request
	Header          *RevelHeader    // The revel header
	ContentType     string          // The content type
	Format          string          // The output format "html", "xml", "json", or "txt"
	AcceptLanguages AcceptLanguages // The languages to accept
	Locale          string          // THe locale
	WebSocket       ServerWebSocket // The websocket
	Method          string          // The method
	RemoteAddr      string          // The remote address
	Host            string          // The host
	// URL request path from the server (built)
	URL *url.URL // The url
	// DEPRECATED use GetForm()
	Form url.Values // The Form
	// DEPRECATED use GetMultipartForm()
	MultipartForm *MultipartForm // The multipart form
	controller    *Controller    // The controller, so some of this data can be fetched
}

var FORM_NOT_FOUND = errors.New("Form Not Found")
var httpLog = RevelLog.New("section", "http")

// Response is Revel's HTTP response object structure
type Response struct {
	Status      int
	ContentType string
	Out         OutResponse
	writer      io.Writer
}

// The output response
type OutResponse struct {
	// internalHeader.Server Set by ServerResponse.Get(HTTP_SERVER_HEADER), saves calling the get every time the header needs to be written to
	internalHeader *RevelHeader   // The internal header
	Server         ServerResponse // The server response
	response       *Response      // The response
}

// The header defined in Revel
type RevelHeader struct {
	Server ServerHeader // The server
}

// NewResponse wraps ServerResponse inside a Revel's Response and returns it
func NewResponse(w ServerResponse) (r *Response) {
	r = &Response{Out: OutResponse{Server: w, internalHeader: &RevelHeader{}}}
	r.Out.response = r
	return r
}

// NewRequest returns a Revel's HTTP request instance with given HTTP instance
func NewRequest(r ServerRequest) *Request {
	req := &Request{Header: &RevelHeader{}}
	if r != nil {
		req.SetRequest(r)
	}
	return req
}
func (req *Request) SetRequest(r ServerRequest) {
	req.In = r
	if h, e := req.In.Get(HTTP_SERVER_HEADER); e == nil {
		req.Header.Server = h.(ServerHeader)
	}

	req.URL, _ = req.GetValue(HTTP_URL).(*url.URL)
	req.ContentType = ResolveContentType(req)
	req.Format = ResolveFormat(req)
	req.AcceptLanguages = ResolveAcceptLanguage(req)
	req.Method, _ = req.GetValue(HTTP_METHOD).(string)
	req.RemoteAddr, _ = req.GetValue(HTTP_REMOTE_ADDR).(string)
	req.Host, _ = req.GetValue(HTTP_HOST).(string)

}

// Returns a cookie
func (req *Request) Cookie(key string) (ServerCookie, error) {
	if req.Header.Server != nil {
		return req.Header.Server.GetCookie(key)
	}
	return nil, http.ErrNoCookie
}

// Fetch the requested URI
func (req *Request) GetRequestURI() string {
	uri, _ := req.GetValue(HTTP_REQUEST_URI).(string)
	return uri
}

// Fetch the query
func (req *Request) GetQuery() (v url.Values) {
	v, _ = req.GetValue(ENGINE_PARAMETERS).(url.Values)
	return
}

// Fetch the path
func (req *Request) GetPath() (path string) {
	path, _ = req.GetValue(ENGINE_PATH).(string)
	return
}

// Fetch the body
func (req *Request) GetBody() (body io.Reader) {
	body, _ = req.GetValue(HTTP_BODY).(io.Reader)
	return
}

// Fetch the context
func (req *Request) Context() (c context.Context) {
	c, _ = req.GetValue(HTTP_REQUEST_CONTEXT).(context.Context)
	return
}

// Deprecated use controller.Params.Get()
func (req *Request) FormValue(key string) (value string) {
	return req.controller.Params.Get(key)
}

// Deprecated use controller.Params.Form[Key]
func (req *Request) PostFormValue(key string) (value string) {
	valueList := req.controller.Params.Form[key]
	if len(valueList) > 0 {
		value = valueList[0]
	}
	return
}

// Deprecated use GetForm() instead
func (req *Request) ParseForm() (e error) {
	if req.Form == nil {
		req.Form, e = req.GetForm()
	}
	return
}

func (req *Request) GetForm() (url.Values, error) {
	if form, err := req.In.Get(HTTP_FORM); err != nil {
		return nil, err
	} else if values, found := form.(url.Values); found {
		req.Form = values
		return values, nil
	}
	return nil, FORM_NOT_FOUND
}

// Deprecated for backwards compatibility only
type MultipartForm struct {
	File   map[string][]*multipart.FileHeader
	Value  url.Values
	origin ServerMultipartForm
}

func (req *Request) MultipartReader() (*multipart.Reader, error) {

	return nil, errors.New("MultipartReader not supported, use controller.Param")
}

// Deprecated for backwards compatibility only
func newMultipareForm(s ServerMultipartForm) (f *MultipartForm) {
	return &MultipartForm{File: s.GetFiles(), Value: s.GetValues(), origin: s}
}

// Deprecated use GetMultipartForm() instead
func (req *Request) ParseMultipartForm(_ int64) (e error) {
	var s ServerMultipartForm
	if s, e = req.GetMultipartForm(); e == nil {
		req.MultipartForm = newMultipareForm(s)
	}
	return
}

// Return the args for the controller
func (req *Request) Args() map[string]interface{} {
	return req.controller.Args
}

// Return a multipart form
func (req *Request) GetMultipartForm() (ServerMultipartForm, error) {
	if form, err := req.In.Get(HTTP_MULTIPART_FORM); err != nil {
		return nil, err
	} else if values, found := form.(ServerMultipartForm); found {
		return values, nil
	}
	return nil, FORM_NOT_FOUND
}

// Destroy the request
func (req *Request) Destroy() {
	req.In = nil
	req.ContentType = ""
	req.Format = ""
	req.AcceptLanguages = nil
	req.Method = ""
	req.RemoteAddr = ""
	req.Host = ""
	req.Header.Destroy()
	req.URL = nil
	req.Form = nil
	req.MultipartForm = nil
}

// Set the server response
func (resp *Response) SetResponse(r ServerResponse) {
	resp.Out.Server = r
	if h, e := r.Get(HTTP_SERVER_HEADER); e == nil {
		resp.Out.internalHeader.Server, _ = h.(ServerHeader)
	}
}

// Destroy the output response
func (o *OutResponse) Destroy() {
	o.response = nil
	o.internalHeader.Destroy()
}

// Destroy the RevelHeader
func (h *RevelHeader) Destroy() {
	h.Server = nil
}

// Destroy the Response
func (resp *Response) Destroy() {
	resp.Out.Destroy()
	resp.Status = 0
	resp.ContentType = ""
	resp.writer = nil
}

// UserAgent returns the client's User-Agent header string.
func (r *Request) UserAgent() string {
	return r.Header.Get("User-Agent")
}

// Referer returns the client's Referer header string.
func (req *Request) Referer() string {
	return req.Header.Get("Referer")
}

// Return the httpheader for the key
func (req *Request) GetHttpHeader(key string) string {
	return req.Header.Get(key)
}

// Return the value from the server
func (r *Request) GetValue(key int) (value interface{}) {
	value, _ = r.In.Get(key)
	return
}

// WriteHeader writes the header (for now, just the status code).
// The status may be set directly by the application (c.Response.Status = 501).
// If it isn't, then fall back to the provided status code.
func (resp *Response) WriteHeader(defaultStatusCode int, defaultContentType string) {
	if resp.ContentType == "" {
		resp.ContentType = defaultContentType
	}
	resp.Out.internalHeader.Set("Content-Type", resp.ContentType)
	if resp.Status == 0 {
		resp.Status = defaultStatusCode
	}
	resp.SetStatus(resp.Status)
}
func (resp *Response) SetStatus(statusCode int) {
	if resp.Out.internalHeader.Server != nil {
		resp.Out.internalHeader.Server.SetStatus(statusCode)
	} else {
		resp.Out.Server.Set(ENGINE_RESPONSE_STATUS, statusCode)
	}

}

// Return the writer
func (resp *Response) GetWriter() (writer io.Writer) {
	writer = resp.writer
	if writer == nil {
		if w, e := resp.Out.Server.Get(ENGINE_WRITER); e == nil {
			writer, resp.writer = w.(io.Writer), w.(io.Writer)
		}
	}

	return
}

// Replace the writer
func (resp *Response) SetWriter(writer io.Writer) bool {
	resp.writer = writer
	// Leave it up to the engine to flush and close the writer
	return resp.Out.Server.Set(ENGINE_WRITER, writer)
}

// Passes full control to the response to the caller - terminates any initial writes
func (resp *Response) GetStreamWriter() (writer StreamWriter) {
	if w, e := resp.Out.Server.Get(HTTP_STREAM_WRITER); e == nil {
		writer = w.(StreamWriter)
	}
	return
}

// Return the header
func (o *OutResponse) Header() *RevelHeader {
	return o.internalHeader
}

// Write the header out
func (o *OutResponse) Write(data []byte) (int, error) {
	return o.response.GetWriter().Write(data)
}

// Set a value in the header
func (h *RevelHeader) Set(key, value string) {
	if h.Server != nil {
		h.Server.Set(key, value)
	}
}

// Add a key to the header
func (h *RevelHeader) Add(key, value string) {
	if h.Server != nil {
		h.Server.Add(key, value)
	}
}

// Set a cookie in the header
func (h *RevelHeader) SetCookie(cookie string) {
	if h.Server != nil {
		h.Server.SetCookie(cookie)
	}
}

// Set the status for the header
func (h *RevelHeader) SetStatus(status int) {
	if h.Server != nil {
		h.Server.SetStatus(status)
	}
}

// Get a key from the header
func (h *RevelHeader) Get(key string) (value string) {
	values := h.GetAll(key)
	if len(values) > 0 {
		value = values[0]
	}
	return
}

// GetAll returns []string of items (the header split by a comma)
func (h *RevelHeader) GetAll(key string) (values []string) {
	if h.Server != nil {
		values = h.Server.Get(key)
	}
	return
}

// ResolveContentType gets the content type.
// e.g. From "multipart/form-data; boundary=--" to "multipart/form-data"
// If none is specified, returns "text/html" by default.
func ResolveContentType(req *Request) string {

	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		return "text/html"
	}
	return strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
}

// ResolveFormat maps the request's Accept MIME type declaration to
// a Request.Format attribute, specifically "html", "xml", "json", or "txt",
// returning a default of "html" when Accept header cannot be mapped to a
// value above.
func ResolveFormat(req *Request) string {
	ext := strings.ToLower(filepath.Ext(req.GetPath()))
	switch ext {
	case ".html":
		return "html"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".txt":
		return "txt"
	}

	accept := req.GetHttpHeader("accept")

	switch {
	case accept == "",
		strings.HasPrefix(accept, "*/*"), // */
		strings.Contains(accept, "application/xhtml"),
		strings.Contains(accept, "text/html"):
		return "html"
	case strings.Contains(accept, "application/json"),
		strings.Contains(accept, "text/javascript"),
		strings.Contains(accept, "application/javascript"):
		return "json"
	case strings.Contains(accept, "application/xml"),
		strings.Contains(accept, "text/xml"):
		return "xml"
	case strings.Contains(accept, "text/plain"):
		return "txt"
	}

	return "html"
}

// AcceptLanguage is a single language from the Accept-Language HTTP header.
type AcceptLanguage struct {
	Language string
	Quality  float32
}

// AcceptLanguages is collection of sortable AcceptLanguage instances.
type AcceptLanguages []AcceptLanguage

func (al AcceptLanguages) Len() int           { return len(al) }
func (al AcceptLanguages) Swap(i, j int)      { al[i], al[j] = al[j], al[i] }
func (al AcceptLanguages) Less(i, j int) bool { return al[i].Quality > al[j].Quality }
func (al AcceptLanguages) String() string {
	output := bytes.NewBufferString("")
	for i, language := range al {
		if _, err := output.WriteString(fmt.Sprintf("%s (%1.1f)", language.Language, language.Quality)); err != nil {
			httpLog.Error("String: WriteString failed:", "error", err)
		}
		if i != len(al)-1 {
			if _, err := output.WriteString(", "); err != nil {
				httpLog.Error("String: WriteString failed:", "error", err)
			}
		}
	}
	return output.String()
}

// ResolveAcceptLanguage returns a sorted list of Accept-Language
// header values.
//
// The results are sorted using the quality defined in the header for each
// language range with the most qualified language range as the first
// element in the slice.
//
// See the HTTP header fields specification
// (http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.4) for more details.
func ResolveAcceptLanguage(req *Request) AcceptLanguages {
	header := req.Header.Get("Accept-Language")
	if header == "" {
		return req.AcceptLanguages
	}

	acceptLanguageHeaderValues := strings.Split(header, ",")
	acceptLanguages := make(AcceptLanguages, len(acceptLanguageHeaderValues))

	for i, languageRange := range acceptLanguageHeaderValues {
		if qualifiedRange := strings.Split(languageRange, ";q="); len(qualifiedRange) == 2 {
			quality, err := strconv.ParseFloat(qualifiedRange[1], 32)
			if err != nil {
				httpLog.Warn("Detected malformed Accept-Language header quality in  assuming quality is 1", "languageRange", languageRange)
				acceptLanguages[i] = AcceptLanguage{qualifiedRange[0], 1}
			} else {
				acceptLanguages[i] = AcceptLanguage{qualifiedRange[0], float32(quality)}
			}
		} else {
			acceptLanguages[i] = AcceptLanguage{languageRange, 1}
		}
	}

	sort.Sort(acceptLanguages)
	return acceptLanguages
}
