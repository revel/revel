package revel

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type TestSuite struct {
	Client       *http.Client
	Response     *http.Response
	ResponseBody []byte
}

var TestSuites []interface{} // Array of structs that embed TestSuite

// NewTestSuite returns an initialized TestSuite ready for use. It is invoked
// by the test harness to initialize the embedded field in application tests.
func NewTestSuite() TestSuite {
	return TestSuite{Client: &http.Client{}}
}

// Return the base URL of the server, e.g. "http://127.0.0.1:8557"
func (t *TestSuite) BaseUrl() string {
	if Server.Addr[0] == ':' {
		return "http://127.0.0.1" + Server.Addr
	}
	return "http://" + Server.Addr
}

// Issue a GET request to the given path and store the result in Request and
// RequestBody.
func (t *TestSuite) Get(path string) {
	req, err := http.NewRequest("GET", t.BaseUrl()+path, nil)
	if err != nil {
		panic(err)
	}
	t.MakeRequest(req)
}

// Issue a POST request to the given path, sending the given Content-Type and
// data, and store the result in Request and RequestBody.  "data" may be nil.
func (t *TestSuite) Post(path string, contentType string, reader io.Reader) {
	req, err := http.NewRequest("POST", t.BaseUrl()+path, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", contentType)
	t.MakeRequest(req)
}

// Issue a POST request to the given path as a form post of the given key and
// values, and store the result in Request and RequestBody.  
func (t *TestSuite) PostForm(path string, data url.Values) {
	t.Post(path, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// Issue any request and read the response. If successful, the caller may
// examine the Response and ResponseBody properties.
func (t *TestSuite) MakeRequest(req *http.Request) {
	var err error
	if t.Response, err = t.Client.Do(req); err != nil {
		panic(err)
	}
	if t.ResponseBody, err = ioutil.ReadAll(t.Response.Body); err != nil {
		panic(err)
	}
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

func (t *TestSuite) AssertEqual(expected interface{}, actual interface{}) {
	if(expected != actual) {
		panic(fmt.Errorf("(expected) %v != %v (actual)", expected, actual))
	}
}

func (t *TestSuite) Assert(exp bool) {
	t.Assertf(exp, "Assertion failed")
}

func (t *TestSuite) Assertf(exp bool, formatStr string, args ...interface{}) {
	if !exp {
		panic(fmt.Errorf(formatStr, args))
	}
}
