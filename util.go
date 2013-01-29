package revel

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"reflect"
	"regexp"
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

var (
	cookieKeyValueParser = regexp.MustCompile("\x00([^:]*):([^\x00]*)\x00")
)

// Takes the raw (escaped) cookie value and parses out key values.
func ParseKeyValueCookie(val string, cb func(key, val string)) {
	val, _ = url.QueryUnescape(val)
	if matches := cookieKeyValueParser.FindAllStringSubmatch(val, -1); matches != nil {
		for _, match := range matches {
			cb(match[1], match[2])
		}
	}
}

const DefaultFileContentType = "application/octet-stream"

var mimeConfig *MergedConfig

// Load mime-types.conf on init.
func loadMimeConfig() {
	var err error
	mimeConfig, err = LoadConfig("mime-types.conf")
	if err != nil {
		ERROR.Fatalln("Failed to load mime type config:", err)
	}
}

func init() {
	InitHooks = append(InitHooks, loadMimeConfig)
}

// Returns a MIME content type based on the filename's extension.
// If no appropriate one is found, returns "application/octet-stream" by default.
// Additionally, specifies the charset as UTF-8 for text/* types.
func ContentTypeByFilename(filename string) string {
	dot := strings.LastIndex(filename, ".")
	if dot == -1 || dot+1 >= len(filename) {
		return DefaultFileContentType
	}

	extension := filename[dot+1:]
	contentType := mimeConfig.StringDefault(extension, "")
	if contentType == "" {
		return DefaultFileContentType
	}

	if strings.HasPrefix(contentType, "text/") {
		return contentType + "; charset=utf-8"
	}

	return contentType
}

// DirExists returns true if the given path exists and is a directory.
func DirExists(filename string) bool {
	fileInfo, err := os.Stat(filename)
	return err == nil && fileInfo.IsDir()
}

func FirstNonEmpty(strs ...string) string {
	for _, str := range strs {
		if len(str) > 0 {
			return str
		}
	}
	return ""
}
