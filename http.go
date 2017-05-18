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
)

// Request Revel's HTTP request object structure
type Request struct {
	In              ServerRequest
	Header          *RevelHeader
	ContentType     string
	Format          string // "html", "xml", "json", or "txt"
	AcceptLanguages AcceptLanguages
	Locale          string
	Websocket       ServerWebSocket
	Method          string
	RemoteAddr      string
	Host            string
}

var FORM_NOT_FOUND = errors.New("Form Not Found")

// Response Revel's HTTP response object structure
type Response struct {
	Status      int
	ContentType string
	Out         OutResponse
	writer      io.Writer
}
type OutResponse struct {
	internalHeader *RevelHeader
	Server         ServerResponse
	response       *Response
}

type RevelHeader struct {
	Server ServerHeader
}

// NewResponse returns a Revel's HTTP response instance with given instance
func NewResponse(w ServerResponse) (r *Response) {
	r = &Response{Out: OutResponse{Server: w, internalHeader: &RevelHeader{}}}
	r.Out.response = r
	return r
}

// NewRequest returns a Revel's HTTP request instance with given HTTP instance
func NewRequest(r ServerRequest) *Request {
	req := &Request{Header:&RevelHeader{}}
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
	req.ContentType = ResolveContentType(req)
	req.Format = ResolveFormat(req)
	req.AcceptLanguages = ResolveAcceptLanguage(req)
	req.Method, _ = req.GetValue(HTTP_METHOD).(string)
	req.RemoteAddr, _ = req.GetValue(HTTP_REMOTE_ADDR).(string)
	req.Host, _ = req.GetValue(HTTP_HOST).(string)

}
func (req *Request) Cookie(key string) (ServerCookie, error) {
	if req.Header.Server != nil {
		return req.Header.Server.GetCookie(key)
	}
	return nil, http.ErrNoCookie
}

func (req *Request) GetRequestURI() string {
	uri, _ := req.GetValue(HTTP_REQUEST_URI).(string)
	return uri
}
func (req *Request) GetQuery() (v url.Values) {
	v, _ = req.GetValue(ENGINE_PARAMETERS).(url.Values)
	return
}
func (req *Request) GetPath() (path string) {
	path, _ = req.GetValue(ENGINE_PATH).(string)
	return
}
func (req *Request) GetBody() (body io.Reader) {
	body, _ = req.GetValue(HTTP_BODY).(io.Reader)
	return
}

func (req *Request) GetForm() (url.Values, error) {
	if form, err := req.In.Get(HTTP_FORM); err != nil {
		return nil, nil
	} else if values, found := form.(url.Values); found {
		return values, nil
	}
	return nil, FORM_NOT_FOUND
}
func (req *Request) GetMultipartForm() (ServerMultipartForm, error) {
	if form, err := req.In.Get(HTTP_MULTIPART_FORM); err != nil {
		return nil, nil
	} else if values, found := form.(ServerMultipartForm); found {
		return values, nil
	}
	return nil, FORM_NOT_FOUND
}
func (req *Request) Destroy() {
	req.In = nil
	req.ContentType = ""
	req.Format = ""
	req.AcceptLanguages = nil
	req.Method = ""
	req.RemoteAddr = ""
	req.Host = ""
	req.Header.Destroy()
}
func (resp *Response) SetResponse(r ServerResponse) {
	resp.Out.Server = r
	if h, e := r.Get(HTTP_SERVER_HEADER); e == nil {
		resp.Out.internalHeader.Server, _ = h.(ServerHeader)
	}
}
func (o *OutResponse) Destroy() {
	o.response = nil
	o.internalHeader.Destroy()
}
func (h *RevelHeader) Destroy() {
	h.Server = nil
}
func (resp *Response) Destroy() {
	resp.Out.Destroy()
	resp.Status = 0
	resp.ContentType = ""
	resp.writer = nil
}

// UserAgent returns the client's User-Agent, if sent in the request.
func (r *Request) UserAgent() string {
	return r.Header.Get("User-Agent")
}

func (r *Request) Referer() string {
	return r.Header.Get("Referer")
}
func (r *Request) GetHttpHeader(key string) string {
	return r.Header.Get("Referer")
}

func (r *Request) GetValue(key int) (value interface{}) {
	value, _ = r.In.Get(key)
	return
}

// WriteHeader writes the header (for now, just the status code).
// The status may be set directly by the application (c.Response.Status = 501).
// if it isn't, then fall back to the provided status code.
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
	if resp.Out.internalHeader.Server == nil {
		resp.Out.internalHeader.Server.SetStatus(statusCode)
	} else {
		resp.Out.Server.Set(ENGINE_RESPONSE_STATUS, statusCode)
	}

}
func (resp *Response) GetWriter() (writer io.Writer) {
	writer = resp.writer
	if writer == nil {
		if w, e := resp.Out.Server.Get(ENGINE_WRITER); e == nil {
			writer, resp.writer = w.(io.Writer), w.(io.Writer)
		}
	}
	return
}
func (resp *Response) SetWriter(writer io.Writer) bool {
	resp.writer = writer
	// Leave it up to the engine to flush and close the writer
	return resp.Out.Server.Set(ENGINE_WRITER, writer)
}
func (resp *Response) GetStreamWriter() (writer StreamWriter) {
	if w, e := resp.Out.Server.Get(HTTP_STREAM_WRITER); e == nil {
		writer = w.(StreamWriter)
	}
	return
}
func (o *OutResponse) Header() *RevelHeader {
	return o.internalHeader
}
func (o *OutResponse) Write(data []byte) (int, error) {
	return o.response.GetWriter().Write(data)
}
func (h *RevelHeader) Set(key, value string) {
	if h.Server != nil {
		h.Server.Set(key, value)
	}
}
func (h *RevelHeader) Add(key, value string) {
	if h.Server != nil {
		h.Server.Add(key, value)
	}
}
func (h *RevelHeader) SetCookie(cookie string) {
	if h.Server != nil {
		h.Server.SetCookie(cookie)
	}
}
func (h *RevelHeader) SetStatus(status int) {
	if h.Server != nil {
		h.Server.SetStatus(status)
	}
}

//
func (h *RevelHeader) Get(key string) (value string) {
	values := h.GetAll(key)
	if len(values) > 0 {
		value = values[0]
	}
	return
}

// Return []string of items (the header split by a comma)
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
	accept := req.Header.Get("accept")

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
			ERROR.Println("WriteString failed:", err)
		}
		if i != len(al)-1 {
			if _, err := output.WriteString(", "); err != nil {
				ERROR.Println("WriteString failed:", err)
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
		return nil
	}

	acceptLanguageHeaderValues := strings.Split(header, ",")
	acceptLanguages := make(AcceptLanguages, len(acceptLanguageHeaderValues))

	for i, languageRange := range acceptLanguageHeaderValues {
		if qualifiedRange := strings.Split(languageRange, ";q="); len(qualifiedRange) == 2 {
			quality, err := strconv.ParseFloat(qualifiedRange[1], 32)
			if err != nil {
				WARN.Printf("Detected malformed Accept-Language header quality in '%s', assuming quality is 1", languageRange)
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
