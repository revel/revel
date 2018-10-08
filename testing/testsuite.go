// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package testing

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/revel/revel"

	"github.com/revel/revel/session"
	"golang.org/x/net/websocket"
	"net/http/httptest"
)

type TestSuite struct {
	Client        *http.Client
	Response      *http.Response
	ResponseBody  []byte
	Session       session.Session
	SessionEngine revel.SessionEngine
}

type TestRequest struct {
	*http.Request
	testSuite *TestSuite
}

// This is populated by the generated code in the run/run/go file
var TestSuites []interface{} // Array of structs that embed TestSuite

// NewTestSuite returns an initialized TestSuite ready for use. It is invoked
// by the test harness to initialize the embedded field in application tests.
func NewTestSuite() TestSuite {
	return NewTestSuiteEngine(revel.NewSessionCookieEngine())
}

// Define a new test suite with a custom session engine
func NewTestSuiteEngine(engine revel.SessionEngine) TestSuite {
	jar, _ := cookiejar.New(nil)
	ts := TestSuite{
		Client:        &http.Client{Jar: jar},
		Session:       session.NewSession(),
		SessionEngine: engine,
	}

	return ts
}

// NewTestRequest returns an initialized *TestRequest. It is used for extending
// testsuite package making it possibe to define own methods. Example:
//	type MyTestSuite struct {
//		testing.TestSuite
//	}
//
//	func (t *MyTestSuite) PutFormCustom(...) {
//		req := http.NewRequest(...)
//		...
//		return t.NewTestRequest(req)
//	}
func (t *TestSuite) NewTestRequest(req *http.Request) *TestRequest {
	request := &TestRequest{
		Request:   req,
		testSuite: t,
	}
	return request
}

// Host returns the address and port of the server, e.g. "127.0.0.1:8557"
func (t *TestSuite) Host() string {
	if revel.ServerEngineInit.Address[0] == ':' {
		return "127.0.0.1" + revel.ServerEngineInit.Address
	}
	return revel.ServerEngineInit.Address
}

// BaseUrl returns the base http/https URL of the server, e.g. "http://127.0.0.1:8557".
// The scheme is set to https if http.ssl is set to true in the configuration file.
func (t *TestSuite) BaseUrl() string {
	if revel.HTTPSsl {
		return "https://" + t.Host()
	}
	return "http://" + t.Host()
}

// WebSocketUrl returns the base websocket URL of the server, e.g. "ws://127.0.0.1:8557"
func (t *TestSuite) WebSocketUrl() string {
	return "ws://" + t.Host()
}

// Get issues a GET request to the given path and stores the result in Response
// and ResponseBody.
func (t *TestSuite) Get(path string) {
	t.GetCustom(t.BaseUrl() + path).Send()
}

// GetCustom returns a GET request to the given URI in a form of its wrapper.
func (t *TestSuite) GetCustom(uri string) *TestRequest {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		panic(err)
	}
	return t.NewTestRequest(req)
}

// Delete issues a DELETE request to the given path and stores the result in
// Response and ResponseBody.
func (t *TestSuite) Delete(path string) {
	t.DeleteCustom(t.BaseUrl() + path).Send()
}

// DeleteCustom returns a DELETE request to the given URI in a form of its
// wrapper.
func (t *TestSuite) DeleteCustom(uri string) *TestRequest {
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		panic(err)
	}
	return t.NewTestRequest(req)
}

// Put issues a PUT request to the given path, sending the given Content-Type
// and data, storing the result in Response and ResponseBody. "data" may be nil.
func (t *TestSuite) Put(path string, contentType string, reader io.Reader) {
	t.PutCustom(t.BaseUrl()+path, contentType, reader).Send()
}

// PutCustom returns a PUT request to the given URI with specified Content-Type
// and data in a form of wrapper. "data" may be nil.
func (t *TestSuite) PutCustom(uri string, contentType string, reader io.Reader) *TestRequest {
	req, err := http.NewRequest("PUT", uri, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", contentType)
	return t.NewTestRequest(req)
}

// PutForm issues a PUT request to the given path as a form put of the given key
// and values, and stores the result in Response and ResponseBody.
func (t *TestSuite) PutForm(path string, data url.Values) {
	t.PutFormCustom(t.BaseUrl()+path, data).Send()
}

// PutFormCustom returns a PUT request to the given URI as a form put of the
// given key and values. The request is in a form of TestRequest wrapper.
func (t *TestSuite) PutFormCustom(uri string, data url.Values) *TestRequest {
	return t.PutCustom(uri, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// Patch issues a PATCH request to the given path, sending the given
// Content-Type and data, and stores the result in Response and ResponseBody.
// "data" may be nil.
func (t *TestSuite) Patch(path string, contentType string, reader io.Reader) {
	t.PatchCustom(t.BaseUrl()+path, contentType, reader).Send()
}

// PatchCustom returns a PATCH request to the given URI with specified
// Content-Type and data in a form of wrapper. "data" may be nil.
func (t *TestSuite) PatchCustom(uri string, contentType string, reader io.Reader) *TestRequest {
	req, err := http.NewRequest("PATCH", uri, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", contentType)
	return t.NewTestRequest(req)
}

// Post issues a POST request to the given path, sending the given Content-Type
// and data, storing the result in Response and ResponseBody. "data" may be nil.
func (t *TestSuite) Post(path string, contentType string, reader io.Reader) {
	t.PostCustom(t.BaseUrl()+path, contentType, reader).Send()
}

// PostCustom returns a POST request to the given URI with specified
// Content-Type and data in a form of wrapper. "data" may be nil.
func (t *TestSuite) PostCustom(uri string, contentType string, reader io.Reader) *TestRequest {
	req, err := http.NewRequest("POST", uri, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", contentType)
	return t.NewTestRequest(req)
}

// PostForm issues a POST request to the given path as a form post of the given
// key and values, and stores the result in Response and ResponseBody.
func (t *TestSuite) PostForm(path string, data url.Values) {
	t.PostFormCustom(t.BaseUrl()+path, data).Send()
}

// PostFormCustom returns a POST request to the given URI as a form post of the
// given key and values. The request is in a form of TestRequest wrapper.
func (t *TestSuite) PostFormCustom(uri string, data url.Values) *TestRequest {
	return t.PostCustom(uri, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// PostFile issues a multipart request to the given path sending given params
// and files, and stores the result in Response and ResponseBody.
func (t *TestSuite) PostFile(path string, params url.Values, filePaths url.Values) {
	t.PostFileCustom(t.BaseUrl()+path, params, filePaths).Send()
}

// PostFileCustom returns a multipart request to the given URI in a form of its
// wrapper with the given params and files.
func (t *TestSuite) PostFileCustom(uri string, params url.Values, filePaths url.Values) *TestRequest {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, values := range filePaths {
		for _, value := range values {
			createFormFile(writer, key, value)
		}
	}

	for key, values := range params {
		for _, value := range values {
			err := writer.WriteField(key, value)
			t.AssertEqual(nil, err)
		}
	}
	err := writer.Close()
	t.AssertEqual(nil, err)

	return t.PostCustom(uri, writer.FormDataContentType(), body)
}

// Send issues any request and reads the response. If successful, the caller may
// examine the Response and ResponseBody properties. Session data will be
// added.
func (r *TestRequest) Send() {
	writer := httptest.NewRecorder()
	context := revel.NewGoContext(nil)
	context.Request.SetRequest(r.Request)
	context.Response.SetResponse(writer)
	controller := revel.NewController(context)
	controller.Session = r.testSuite.Session

	r.testSuite.SessionEngine.Encode(controller)
	response := http.Response{Header: writer.Header()}
	cookies := response.Cookies()
	for _, c := range cookies {
		r.AddCookie(c)
	}
	r.MakeRequest()
}

// MakeRequest issues any request and read the response. If successful, the
// caller may examine the Response and ResponseBody properties. You will need to
// manage session / cookie data manually
func (r *TestRequest) MakeRequest() {
	var err error
	if r.testSuite.Response, err = r.testSuite.Client.Do(r.Request); err != nil {
		panic(err)
	}
	if r.testSuite.ResponseBody, err = ioutil.ReadAll(r.testSuite.Response.Body); err != nil {
		panic(err)
	}

	// Create the controller again to receive the response for processing.
	context := revel.NewGoContext(nil)
	// Set the request with the header from the response..
	newRequest := &http.Request{URL: r.URL, Header: r.testSuite.Response.Header}
	for _, cookie := range r.testSuite.Client.Jar.Cookies(r.Request.URL) {
		newRequest.AddCookie(cookie)
	}
	context.Request.SetRequest(newRequest)
	context.Response.SetResponse(httptest.NewRecorder())
	controller := revel.NewController(context)

	// Decode the session data from the controller and assign it to the session
	r.testSuite.SessionEngine.Decode(controller)
	r.testSuite.Session = controller.Session
}

// WebSocket creates a websocket connection to the given path and returns it
func (t *TestSuite) WebSocket(path string) *websocket.Conn {
	origin := t.BaseUrl() + "/"
	urlPath := t.WebSocketUrl() + path
	ws, err := websocket.Dial(urlPath, "", origin)
	if err != nil {
		panic(err)
	}
	return ws
}

func (t *TestSuite) AssertOk() {
	t.AssertStatus(http.StatusOK)
}

func (t *TestSuite) AssertNotFound() {
	t.AssertStatus(http.StatusNotFound)
}

func (t *TestSuite) AssertStatus(status int) {
	if t.Response.StatusCode != status {
		panic(fmt.Errorf("Status: (expected) %d != %d (actual)", status, t.Response.StatusCode))
	}
}

func (t *TestSuite) AssertContentType(contentType string) {
	t.AssertHeader("Content-Type", contentType)
}

func (t *TestSuite) AssertHeader(name, value string) {
	actual := t.Response.Header.Get(name)
	if actual != value {
		panic(fmt.Errorf("Header %s: (expected) %s != %s (actual)", name, value, actual))
	}
}

func (t *TestSuite) AssertEqual(expected, actual interface{}) {
	if !revel.Equal(expected, actual) {
		panic(fmt.Errorf("(expected) %v != %v (actual)", expected, actual))
	}
}

func (t *TestSuite) AssertNotEqual(expected, actual interface{}) {
	if revel.Equal(expected, actual) {
		panic(fmt.Errorf("(expected) %v == %v (actual)", expected, actual))
	}
}

func (t *TestSuite) Assert(exp bool) {
	t.Assertf(exp, "Assertion failed")
}

func (t *TestSuite) Assertf(exp bool, formatStr string, args ...interface{}) {
	if !exp {
		panic(fmt.Errorf(formatStr, args...))
	}
}

// AssertContains asserts that the response contains the given string.
func (t *TestSuite) AssertContains(s string) {
	if !bytes.Contains(t.ResponseBody, []byte(s)) {
		panic(fmt.Errorf("Assertion failed. Expected response to contain %s", s))
	}
}

// AssertNotContains asserts that the response does not contain the given string.
func (t *TestSuite) AssertNotContains(s string) {
	if bytes.Contains(t.ResponseBody, []byte(s)) {
		panic(fmt.Errorf("Assertion failed. Expected response not to contain %s", s))
	}
}

// AssertContainsRegex asserts that the response matches the given regular expression.
func (t *TestSuite) AssertContainsRegex(regex string) {
	r := regexp.MustCompile(regex)

	if !r.Match(t.ResponseBody) {
		panic(fmt.Errorf("Assertion failed. Expected response to match regexp %s", regex))
	}
}

func createFormFile(writer *multipart.Writer, fieldname, filename string) {
	// Try to open the file.
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Create a new form-data header with the provided field name and file name.
	// Determine Content-Type of the file by its extension.
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(
		`form-data; name="%s"; filename="%s"`,
		escapeQuotes(fieldname),
		escapeQuotes(filepath.Base(filename)),
	))
	h.Set("Content-Type", "application/octet-stream")
	if ct := mime.TypeByExtension(filepath.Ext(filename)); ct != "" {
		h.Set("Content-Type", ct)
	}
	part, err := writer.CreatePart(h)
	if err != nil {
		panic(err)
	}

	// Copy the content of the file we have opened not reading the whole
	// file into memory.
	_, err = io.Copy(part, file)
	if err != nil {
		panic(err)
	}
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// This function was borrowed from mime/multipart package.
func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
