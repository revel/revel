package rev

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
)

// These tests verify that Controllers are initialized properly, given the range
// of embedding possibilities..

type N struct{ Controller }
type P struct{ *Controller }

type NN struct{ N }
type NP struct{ *N }
type PN struct{ P }
type PP struct{ *P }

type NNN struct{ NN }
type NPN struct{ NP }
type PNP struct{ *PN }
type PPP struct{ *PP }

var GENERATIONS = [][]interface{}{
	{N{}, P{}},
	{NN{}, NP{}, PN{}, PP{}},
	{NNN{}, NPN{}, PNP{}, PPP{}},
}

// This test constructs a bunch of hypothetical app controllers, and verifies
// that the embedded Controller field was set correctly.
func TestNewAppController(t *testing.T) {
	controller := &Controller{Name: "Test"}
	for gen, structs := range GENERATIONS {
		for _, st := range structs {
			typ := reflect.TypeOf(st)
			val := initNewAppController(typ, controller)

			// Drill into the embedded fields to get to the Controller.
			for i := 0; i < gen+1; i++ {
				if val.Kind() == reflect.Ptr {
					val = val.Elem()
				}
				val = val.Field(0)
			}

			var name string
			if val.Type().Kind() == reflect.Ptr {
				name = val.Interface().(*Controller).Name
			} else {
				name = val.Interface().(Controller).Name
			}

			if name != "Test" {
				t.Error("Fail: " + typ.String())
			}
		}
	}
}

// Since the test machinery that goes through all the structs is non-trivial,
// have one redundant test that covers just one complicated case but is dead
// simple.
func TestNewAppController2(t *testing.T) {
	val := initNewAppController(reflect.TypeOf(PNP{}), &Controller{Name: "Test"})
	if val.Interface().(*PNP).PN.P.Controller.Name != "Test" {
		t.Error("PNP not initialized.")
	}
}

// Params: Testing Multipart forms

const (
	MULTIPART_BOUNDARY  = "A"
	MULTIPART_FORM_DATA = `--A
Content-Disposition: form-data; name="text1"

data1
--A
Content-Disposition: form-data; name="text2"

data2
--A
Content-Disposition: form-data; name="text2"

data3
--A
Content-Disposition: form-data; name="file1"; filename="test.txt"
Content-Type: text/plain

content1
--A
Content-Disposition: form-data; name="file2[]"; filename="test.txt"
Content-Type: text/plain

content2
--A
Content-Disposition: form-data; name="file2[]"; filename="favicon.ico"
Content-Type: image/x-icon

xyz
--A
Content-Disposition: form-data; name="file3[0]"; filename="test.txt"
Content-Type: text/plain

content3
--A
Content-Disposition: form-data; name="file3[1]"; filename="favicon.ico"
Content-Type: image/x-icon

zzz
--A--
`
)

// The values represented by the form data.
type fh struct {
	filename string
	content  []byte
}

var (
	expectedValues = map[string][]string{
		"text1": {"data1"},
		"text2": {"data2", "data3"},
	}
	expectedFiles = map[string][]fh{
		"file1":    {fh{"test.txt", []byte("content1")}},
		"file2[]":  {fh{"test.txt", []byte("content2")}, fh{"favicon.ico", []byte("xyz")}},
		"file3[0]": {fh{"test.txt", []byte("content3")}},
		"file3[1]": {fh{"favicon.ico", []byte("zzz")}},
	}
)

func getMultipartRequest() *http.Request {
	req, _ := http.NewRequest("POST", "http://localhost/path",
		bytes.NewBufferString(MULTIPART_FORM_DATA))
	req.Header.Set(
		"Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", MULTIPART_BOUNDARY))
	req.Header.Set(
		"Content-Length", fmt.Sprintf("%d", len(MULTIPART_FORM_DATA)))
	return req
}

func TestMultipartForm(t *testing.T) {
	params := ParseParams(NewRequest(getMultipartRequest()))

	if !reflect.DeepEqual(expectedValues, map[string][]string(params.Values)) {
		t.Errorf("Param values: (expected) %v != %v (actual)",
			expectedValues, map[string][]string(params.Values))
	}

	actualFiles := make(map[string][]fh)
	for key, fileHeaders := range params.Files {
		for _, fileHeader := range fileHeaders {
			file, _ := fileHeader.Open()
			content, _ := ioutil.ReadAll(file)
			actualFiles[key] = append(actualFiles[key], fh{fileHeader.Filename, content})
		}
	}

	if !reflect.DeepEqual(expectedFiles, actualFiles) {
		t.Errorf("Param files: (expected) %v != %v (actual)", expectedFiles, actualFiles)
	}
}

func TestResolveAcceptLanguage(t *testing.T) {
	request := buildHttpRequestWithAcceptLanguage("")
	if result := ResolveAcceptLanguage(request); result != nil {
		t.Errorf("Expected Accept-Language to resolve to an empty string but it was '%s'", result)
	}

	request = buildHttpRequestWithAcceptLanguage("en-GB,en;q=0.8,nl;q=0.6")
	if result := ResolveAcceptLanguage(request); len(result) != 3 || result[0] != "en-GB" || result[1] != "en;q=0.8" || result[2] != "nl;q=0.6" {
		t.Error("Unexpected Accept-Language value after resolve")
	}

	request = buildHttpRequestWithAcceptLanguage("en-GB, en;q=0.8, nl;q=0.6")
	if result := ResolveAcceptLanguage(request); len(result) != 3 || result[0] != "en-GB" || result[1] != "en;q=0.8" || result[2] != "nl;q=0.6" {
		t.Error("Unexpected Accept-Language value after resolve")
	}
}

func buildHttpRequestWithAcceptLanguage(acceptLanguage string) *http.Request {
	request, _ := http.NewRequest("POST", "http://localhost/path", nil)
	request.Header.Set("Accept-Language", acceptLanguage)
	return request
}
