package rev

import (
	"fmt"
	"go/ast"
	"io/ioutil"
	"net/http"
	"reflect"
)

type UnitTest struct{}

type FunctionalTest struct {
	Client       http.Client
	Response     *http.Response
	ResponseBody []byte
}

var (
	UnitTests       []interface{}
	FunctionalTests []interface{} // Array of structs that embed FunctionalTest
)

func (t FunctionalTest) GetPath(url string) {
	var err error
	if t.Response, err = t.Client.Get(Server.Addr + url); err != nil {
		panic(err)
	}
	if t.ResponseBody, err = ioutil.ReadAll(t.Response.Body); err != nil {
		panic(err)
	}
}

func (t FunctionalTest) AssertOk() {
	t.AssertStatus(http.StatusOK)
}

func (t FunctionalTest) AssertStatus(status int) {
	if t.Response.StatusCode != status {
		panic(fmt.Errorf("Status: (expected) %d != %d (actual)", status, t.Response.StatusCode))
	}
}

func (t FunctionalTest) AssertContentType(contentType string) {
	rct := t.Response.Header.Get("Content-Type")
	if rct != contentType {
		panic(fmt.Errorf("Content Type: (expected) %d != %d (actual)", contentType, rct))
	}
}

// Run every method on the given value that meets the following criteria:
// 1. It takes no parameters.
// 2. It returns no values.
// 3. It is exported.
func RunTestSuite(suite interface{}) (succeeded, failed int) {
	v := reflect.ValueOf(suite)
	t := v.Type()
	fmt.Println("Functional test suite:", t.Name())
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		if ast.IsExported(m.Name) && m.Type.NumIn() == 0 && m.Type.NumOut() == 0 {
			fmt.Println(m.Name)
			func() {
				defer func() {
					if err := recover(); err != nil {
						fmt.Println("Fail:", err)
						failed++
					} else {
						succeeded++
					}
				}()
				m.Func.Call([]reflect.Value{v})
			}()
			fmt.Println("OK")
		}
	}
	return
}
