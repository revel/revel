// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package testing

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/revel/revel"
)

func TestMisc(t *testing.T) {
	testSuite := createNewTestSuite(t)

	// test Host value
	if !strings.EqualFold("127.0.0.1:9001", testSuite.Host()) {
		t.Error("Incorrect Host value found.")
	}

	// test BaseUrl
	if !strings.EqualFold("http://127.0.0.1:9001", testSuite.BaseUrl()) {
		t.Error("Incorrect BaseUrl http value found.")
	}
	revel.HTTPSsl = true
	if !strings.EqualFold("https://127.0.0.1:9001", testSuite.BaseUrl()) {
		t.Error("Incorrect BaseUrl https value found.")
	}
	revel.HTTPSsl = false

	// test WebSocketUrl
	if !strings.EqualFold("ws://127.0.0.1:9001", testSuite.WebSocketUrl()) {
		t.Error("Incorrect WebSocketUrl value found.")
	}

	testSuite.AssertNotEqual("Yes", "No")
	testSuite.Assert(true)
}

func TestGet(t *testing.T) {
	ts := createTestServer(testHandle)
	defer ts.Close()

	testSuite := createNewTestSuite(t)

	testSuite.Get("/")
	testSuite.AssertOk()
	testSuite.AssertContains("this is testcase homepage")
	testSuite.AssertNotContains("not exists")
}

func TestGetNotFound(t *testing.T) {
	ts := createTestServer(testHandle)
	defer ts.Close()

	testSuite := createNewTestSuite(t)

	testSuite.Get("/notfound")
	testSuite.AssertNotFound()
	// testSuite.AssertContains("this is testcase homepage")
	// testSuite.AssertNotContains("not exists")
}

func TestGetCustom(t *testing.T) {
	testSuite := createNewTestSuite(t)
	testSuite.GetCustom("http://httpbin.org/get").Send()

	testSuite.AssertOk()
	testSuite.AssertContentType("application/json")
	testSuite.AssertContains("httpbin.org")
	testSuite.AssertContainsRegex("gzip|deflate")
}

func TestDelete(t *testing.T) {
	ts := createTestServer(testHandle)
	defer ts.Close()

	testSuite := createNewTestSuite(t)

	testSuite.Delete("/purchases/10001")
	testSuite.AssertOk()
}

func TestPut(t *testing.T) {
	ts := createTestServer(testHandle)
	defer ts.Close()

	testSuite := createNewTestSuite(t)

	testSuite.Put("/purchases/10002",
		"application/json",
		bytes.NewReader([]byte(`{"sku":"163645GHT", "desc":"This is test product"}`)),
	)
	testSuite.AssertStatus(http.StatusNoContent)
}

func TestPutForm(t *testing.T) {
	ts := createTestServer(testHandle)
	defer ts.Close()

	testSuite := createNewTestSuite(t)

	data := url.Values{}
	data.Add("name", "beacon1name")
	data.Add("value", "beacon1value")

	testSuite.PutForm("/send", data)
	testSuite.AssertStatus(http.StatusNoContent)
}

func TestPatch(t *testing.T) {
	ts := createTestServer(testHandle)
	defer ts.Close()

	testSuite := createNewTestSuite(t)

	testSuite.Patch("/purchases/10003",
		"application/json",
		bytes.NewReader([]byte(`{"desc": "This is test patch for product"}`)),
	)
	testSuite.AssertStatus(http.StatusNoContent)
}

func TestPost(t *testing.T) {
	ts := createTestServer(testHandle)
	defer ts.Close()

	testSuite := createNewTestSuite(t)
	fmt.Println(testSuite.Session.Cookie().Name)

	testSuite.Post("/login",
		"application/json",
		bytes.NewReader([]byte(`{"username":"testuser", "password":"testpass"}`)),
	)
	testSuite.AssertOk()
	testSuite.AssertContains("login successful")
}

func TestPostForm(t *testing.T) {
	ts := createTestServer(testHandle)
	defer ts.Close()

	testSuite := createNewTestSuite(t)

	data := url.Values{}
	data.Add("username", "testuser")
	data.Add("password", "testpassword")

	testSuite.PostForm("/login", data)
	testSuite.AssertOk()
	testSuite.AssertContains("login successful")
}

func TestPostFileUpload(t *testing.T) {
	ts := createTestServer(testHandle)
	defer ts.Close()

	testSuite := createNewTestSuite(t)

	params := url.Values{}
	params.Add("first_name", "Jeevanandam")
	params.Add("last_name", "M.")

	currentDir, _ := os.Getwd()
	basePath := filepath.Dir(currentDir)

	filePaths := url.Values{}
	filePaths.Add("revel_file", filepath.Join(basePath, "revel.go"))
	filePaths.Add("server_file", filepath.Join(basePath, "server.go"))
	filePaths.Add("readme_file", filepath.Join(basePath, "README.md"))

	testSuite.PostFile("/upload", params, filePaths)

	testSuite.AssertOk()
	testSuite.AssertContains("File: revel.go")
	testSuite.AssertContains("File: server.go")
	testSuite.AssertNotContains("File: not_exists.go")
	testSuite.AssertEqual("text/plain; charset=utf-8", testSuite.Response.Header.Get("Content-Type"))

}

func createNewTestSuite(t *testing.T) *TestSuite {
	suite := NewTestSuite()

	if suite.Client == nil || suite.Session == nil {
		t.Error("Unable to create a testsuite")
	}

	return &suite
}

func testHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if r.URL.Path == "/" {
			_, _ = w.Write([]byte(`this is testcase homepage`))
			return
		}
	}

	if r.Method == "POST" {
		if r.URL.Path == "/login" {
			http.SetCookie(w, &http.Cookie{
				Name:     "_SESSION",
				Value:    "This is simple session value",
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				Expires:  time.Now().Add(time.Minute * 5).UTC(),
			})

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{ "id": "success", "message": "login successful" }`))
			return
		}

		handleFileUpload(w, r)
		return
	}

	if r.Method == "DELETE" {
		if r.URL.Path == "/purchases/10001" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	if r.Method == "PUT" {
		if r.URL.Path == "/purchases/10002" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.URL.Path == "/send" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	if r.Method == "PATCH" {
		if r.URL.Path == "/purchases/10003" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/upload" {
		_ = r.ParseMultipartForm(10e6)
		var buf bytes.Buffer
		for _, fhdrs := range r.MultipartForm.File {
			for _, hdr := range fhdrs {
				dotPos := strings.LastIndex(hdr.Filename, ".")
				fname := fmt.Sprintf("%s-%v%s", hdr.Filename[:dotPos], time.Now().Unix(), hdr.Filename[dotPos:])
				_, _ = buf.WriteString(fmt.Sprintf(
					"Firstname: %v\nLastname: %v\nFile: %v\nHeader: %v\nUploaded as: %v\n",
					r.FormValue("first_name"),
					r.FormValue("last_name"),
					hdr.Filename,
					hdr.Header,
					fname))
			}
		}

		_, _ = w.Write(buf.Bytes())

		return
	}
}

func createTestServer(fn func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	testServer := httptest.NewServer(http.HandlerFunc(fn))
	revel.Server.Addr = testServer.URL[7:]
	return testServer
}

func init() {
	if revel.Server == nil {
		revel.Server = &http.Server{
			Addr: ":9001",
		}
	}
}
