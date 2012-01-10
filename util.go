// Various utility functions to make the standard library a bit easier to deal
// with.

package play

import (
	"bytes"
	"io"
	"io/ioutil"
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
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return strings.Split(string(bytes), "\n")
}

func ContainsString(list []string, target string) bool {
	for _, el := range list {
		if el == target {
			return true
		}
	}
	return false
}
