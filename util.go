// Various utility functions to make the standard library a bit easier to deal
// with.

package play

import (
	"bytes"
	"go/build"
	"io"
	"io/ioutil"
	"path"
	"reflect"
	"strings"
)

// Add some more methods to the default Template.
type ExecutableTemplate interface {
	Execute(io.Writer, interface{}) error
}

// Execute a template and returns the result as a string.
func ExecuteTemplate(tmpl ExecutableTemplate, data interface{}) string {
	var b bytes.Buffer
	tmpl.Execute(&b, data)
	return b.String()
}

// Reads the lines of the given file.  Panics in the case of error.
func MustReadLines(filename string) []string {
	r, err := ReadLines(filename)
	if err != nil {
		panic(err)
	}
	return r
}

// Reads the lines of the given file.  Panics in the case of error.
func ReadLines(filename string) ([]string, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(bytes), "\n"), nil
}

func ContainsString(list []string, target string) bool {
	for _, el := range list {
		if el == target {
			return true
		}
	}
	return false
}

// The Import Path is how we can import its code.
// For example, the sample app resides in src/play/sample, and it must be
// imported as "play/sample/...".  Here, the import path is "play/sample".
// This assumes that the user's app is in a GOPATH, which requires the root of
// the packages to be "src".
func GetImportPath(path string) string {
	srcIndex := strings.Index(path, "src")
	if srcIndex == -1 {
		LOG.Fatalf("App directory (%s) does not appear to be below \"src\". " +
			" I don't know how to import your code.  Please use GOPATH layout.",
			path)
	}
	return path[srcIndex+4:]
}

// Look for a given path in the GOPATHs.  Return it as an absolute path.
// Return empty string if not found.
func FindSource(relPath string) string {
	for _, p := range build.Path {
		if p.HasSrc(relPath) {
			return path.Join(p.SrcDir(), relPath)
		}
	}
	return ""
}

// Return the reflect.Method, given a Receiver type and Func value.
func FindMethod(recvType reflect.Type, funcVal *reflect.Value) *reflect.Method {
	// It is not possible to get the name of the method from the Func.
	// Instead, compare it to each method of the Controller.
	for i := 0; i < recvType.NumMethod(); i++ {
		method := recvType.Method(i)
		if method.Func == *funcVal {
			return &method
		}
	}
	return nil
}
