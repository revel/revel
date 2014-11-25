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

	"golang.org/x/net/websocket"
)

type TestSuite struct {
	Client       *http.Client
	Response     *http.Response
	ResponseBody []byte
	Session      revel.Session
}

type TestRequest struct {
	*http.Request
	testSuite *TestSuite
}

var TestSuites []interface{} // Array of structs that embed TestSuite

// NewTestSuite returns an initialized TestSuite ready for use. It is invoked
// by the test harness to initialize the embedded field in application tests.
func NewTestSuite() TestSuite {
	jar, _ := cookiejar.New(nil)
	return TestSuite{
		Client:  &http.Client{Jar: jar},
		Session: make(revel.Session),
	}
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
	return &TestRequest{
		Request:   req,
		testSuite: t,
	}
}

// Return the address and port of the server, e.g. "127.0.0.1:8557"
func (t *TestSuite) Host() string {
	if revel.Server.Addr[0] == ':' {
		return "127.0.0.1" + revel.Server.Addr
	}
	return revel.Server.Addr
}

// Return the base http/https URL of the server, e.g. "http://127.0.0.1:8557".
// The scheme is set to https if http.ssl is set to true in the configuration file.
func (t *TestSuite) BaseUrl() string {
	if revel.HttpSsl {
		return "https://" + t.Host()
	} else {
		return "http://" + t.Host()
	}
}

// Return the base websocket URL of the server, e.g. "ws://127.0.0.1:8557"
func (t *TestSuite) WebSocketUrl() string {
	return "ws://" + t.Host()
}

// Issue a GET request to the given path and store the result in Response and
// ResponseBody.
func (t *TestSuite) Get(path string) {
	t.GetCustom(t.BaseUrl() + path).Send()
}

// Return a GET request to the given uri in a form of its wrapper.
func (t *TestSuite) GetCustom(uri string) *TestRequest {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		panic(err)
	}
	return t.NewTestRequest(req)
}

// Issue a DELETE request to the given path and store the result in Response and
// ResponseBody.
func (t *TestSuite) Delete(path string) {
	t.DeleteCustom(t.BaseUrl() + path).Send()
}

// Return a DELETE request to the given uri in a form of its wrapper.
func (t *TestSuite) DeleteCustom(uri string) *TestRequest {
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		panic(err)
	}
	return t.NewTestRequest(req)
}

// Issue a PUT request to the given path, sending the given Content-Type and
// data, and store the result in Response and ResponseBody.  "data" may be nil.
func (t *TestSuite) Put(path string, contentType string, reader io.Reader) {
	t.PutCustom(t.BaseUrl()+path, contentType, reader).Send()
}

// Return a PUT request to the given uri with specified Content-Type and data
// in a form of wrapper. "data" may be nil.
func (t *TestSuite) PutCustom(uri string, contentType string, reader io.Reader) *TestRequest {
	req, err := http.NewRequest("PUT", uri, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", contentType)
	return t.NewTestRequest(req)
}

// Issue a PATCH request to the given path, sending the given Content-Type and
// data, and store the result in Response and ResponseBody.  "data" may be nil.
func (t *TestSuite) Patch(path string, contentType string, reader io.Reader) {
	t.PatchCustom(t.BaseUrl()+path, contentType, reader).Send()
}

// Return a PATCH request to the given uri with specified Content-Type and data
// in a form of wrapper. "data" may be nil.
func (t *TestSuite) PatchCustom(uri string, contentType string, reader io.Reader) *TestRequest {
	req, err := http.NewRequest("PATCH", uri, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", contentType)
	return t.NewTestRequest(req)
}

// Issue a POST request to the given path, sending the given Content-Type and
// data, and store the result in Response and ResponseBody.  "data" may be nil.
func (t *TestSuite) Post(path string, contentType string, reader io.Reader) {
	t.PostCustom(t.BaseUrl()+path, contentType, reader).Send()
}

// Return a POST request to the given uri with specified Content-Type and data
// in a form of wrapper. "data" may be nil.
func (t *TestSuite) PostCustom(uri string, contentType string, reader io.Reader) *TestRequest {
	req, err := http.NewRequest("POST", uri, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", contentType)
	return t.NewTestRequest(req)
}

// Issue a POST request to the given path as a form post of the given key and
// values, and store the result in Response and ResponseBody.
func (t *TestSuite) PostForm(path string, data url.Values) {
	t.PostFormCustom(t.BaseUrl()+path, data).Send()
}

// Return a POST request to the given uri as a form post of the given key and values.
// The request is in a form of TestRequest wrapper.
func (t *TestSuite) PostFormCustom(uri string, data url.Values) *TestRequest {
	return t.PostCustom(uri, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// Issue a multipart request to the given path sending given params and files,
// and store the result in Response and ResponseBody.
func (t *TestSuite) PostFile(path string, params url.Values, filePaths url.Values) {
	t.PostFileCustom(t.BaseUrl()+path, params, filePaths).Send()
}

// Return a multipart request to the given uri in a form of its wrapper
// with the given params and files.
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

// Issue any request and read the response. If successful, the caller may
// examine the Response and ResponseBody properties. Session data will be
// added to the request cookies for you.
func (r *TestRequest) Send() {
	r.AddCookie(r.testSuite.Session.Cookie())
	r.MakeRequest()
}

// Issue any request and read the response. If successful, the caller may
// examine the Response and ResponseBody properties. You will need to
// manage session / cookie data manually
func (r *TestRequest) MakeRequest() {
	var err error
	if r.testSuite.Response, err = r.testSuite.Client.Do(r.Request); err != nil {
		panic(err)
	}
	if r.testSuite.ResponseBody, err = ioutil.ReadAll(r.testSuite.Response.Body); err != nil {
		panic(err)
	}

	// Look for a session cookie in the response and parse it.
	sessionCookieName := r.testSuite.Session.Cookie().Name
	for _, cookie := range r.testSuite.Client.Jar.Cookies(r.Request.URL) {
		if cookie.Name == sessionCookieName {
			r.testSuite.Session = revel.GetSessionFromCookie(cookie)
			break
		}
	}
}

// Create a websocket connection to the given path and return the connection
func (t *TestSuite) WebSocket(path string) *websocket.Conn {
	origin := t.BaseUrl() + "/"
	url := t.WebSocketUrl() + path
	ws, err := websocket.Dial(url, "", origin)
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

// Assert that the response contains the given string.
func (t *TestSuite) AssertContains(s string) {
	if !bytes.Contains(t.ResponseBody, []byte(s)) {
		panic(fmt.Errorf("Assertion failed. Expected response to contain %s", s))
	}
}

// Assert that the response does not contain the given string.
func (t *TestSuite) AssertNotContains(s string) {
	if bytes.Contains(t.ResponseBody, []byte(s)) {
		panic(fmt.Errorf("Assertion failed. Expected response not to contain %s", s))
	}
}

// Assert that the response matches the given regular expression.
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
	defer file.Close()

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
