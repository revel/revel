package revel

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

type TestSuite struct {
	Client       *http.Client
	Response     *http.Response
	ResponseBody []byte
	Session      Session
}

var TestSuites []interface{} // Array of structs that embed TestSuite

// NewTestSuite returns an initialized TestSuite ready for use. It is invoked
// by the test harness to initialize the embedded field in application tests.
func NewTestSuite() TestSuite {
	jar, _ := cookiejar.New(nil)
	return TestSuite{
		Client:  &http.Client{Jar: jar},
		Session: make(Session),
	}
}

// Return the address and port of the server, e.g. "127.0.0.1:8557"
func (t *TestSuite) Host() string {
	if Server.Addr[0] == ':' {
		return "127.0.0.1" + Server.Addr
	}
	return Server.Addr
}

// Return the base http/https URL of the server, e.g. "http://127.0.0.1:8557".
// The scheme is set to https if http.ssl is set to true in the configuration file.
func (t *TestSuite) BaseUrl() string {
	if HttpSsl {
		return "https://" + t.Host()
	} else {
		return "http://" + t.Host()
	}
}

// Return the base websocket URL of the server, e.g. "ws://127.0.0.1:8557"
func (t *TestSuite) WebSocketUrl() string {
	return "ws://" + t.Host()
}

// Issue a GET request to the given path and store the result in Request and
// RequestBody.
func (t *TestSuite) Get(path string) {
	req, err := http.NewRequest("GET", t.BaseUrl()+path, nil)
	if err != nil {
		panic(err)
	}
	t.MakeRequestSession(req)
}

// Issue a DELETE request to the given path and store the result in Request and
// RequestBody.
func (t *TestSuite) Delete(path string) {
	req, err := http.NewRequest("DELETE", t.BaseUrl()+path, nil)
	if err != nil {
		panic(err)
	}
	t.MakeRequestSession(req)
}

// Issue a POST request to the given path, sending the given Content-Type and
// data, and store the result in Request and RequestBody.  "data" may be nil.
func (t *TestSuite) Post(path string, contentType string, reader io.Reader) {
	req, err := http.NewRequest("POST", t.BaseUrl()+path, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", contentType)
	t.MakeRequestSession(req)
}

// Issue a POST request to the given path as a form post of the given key and
// values, and store the result in Request and RequestBody.
func (t *TestSuite) PostForm(path string, data url.Values) {
	t.Post(path, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// Issue a multipart request for the method & fields given and read the response.
// If successful, the caller may examine the Response and ResponseBody properties.
func (t *TestSuite) MakeMultipartRequest(method string, path string, fields map[string]string) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for key, value := range fields {
		w.WriteField(key, value)
	}
	w.Close() //adds the terminating boundary

	req, err := http.NewRequest(method, t.BaseUrl()+path, &b)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	t.MakeRequest(req)
}

// Issue any request and read the response. If successful, the caller may
// examine the Response and ResponseBody properties. Session data will be
// added to the request cookies for you.
func (t *TestSuite) MakeRequestSession(req *http.Request) {
	req.AddCookie(t.Session.cookie())
	t.MakeRequest(req)
}

// Issue any request and read the response. If successful, the caller may
// examine the Response and ResponseBody properties. You will need to
// manage session / cookie data manually
func (t *TestSuite) MakeRequest(req *http.Request) {
	var err error
	if t.Response, err = t.Client.Do(req); err != nil {
		panic(err)
	}
	if t.ResponseBody, err = ioutil.ReadAll(t.Response.Body); err != nil {
		panic(err)
	}

	// Look for a session cookie in the response and parse it.
	sessionCookieName := t.Session.cookie().Name
	for _, cookie := range t.Client.Jar.Cookies(req.URL) {
		if cookie.Name == sessionCookieName {
			t.Session = getSessionFromCookie(cookie)
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
	if !Equal(expected, actual) {
		panic(fmt.Errorf("(expected) %v != %v (actual)", expected, actual))
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

// Assert that the response matches the given regular expression.BUG
func (t *TestSuite) AssertContainsRegex(regex string) {
	r := regexp.MustCompile(regex)

	if !r.Match(t.ResponseBody) {
		panic(fmt.Errorf("Assertion failed. Expected response to match regexp %s", regex))
	}
}
