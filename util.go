package revel

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/revel/revel/config"
)

const (
	DefaultFileContentType = "application/octet-stream"
)

var (
	cookieKeyValueParser = regexp.MustCompile("\x00([^:]*):([^\x00]*)\x00")
	hdrForwardedFor      = http.CanonicalHeaderKey("X-Forwarded-For")
	hdrRealIP            = http.CanonicalHeaderKey("X-Real-Ip")

	mimeConfig *config.Context
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
func FindMethod(recvType reflect.Type, funcVal reflect.Value) *reflect.Method {
	// It is not possible to get the name of the method from the Func.
	// Instead, compare it to each method of the Controller.
	for i := 0; i < recvType.NumMethod(); i++ {
		method := recvType.Method(i)
		if method.Func.Pointer() == funcVal.Pointer() {
			return &method
		}
	}
	return nil
}

// Takes the raw (escaped) cookie value and parses out key values.
func ParseKeyValueCookie(val string, cb func(key, val string)) {
	val, _ = url.QueryUnescape(val)
	if matches := cookieKeyValueParser.FindAllStringSubmatch(val, -1); matches != nil {
		for _, match := range matches {
			cb(match[1], match[2])
		}
	}
}

// Load mime-types.conf on init.
func LoadMimeConfig() {
	var err error
	mimeConfig, err = config.LoadContext("mime-types.conf", ConfPaths)
	if err != nil {
		ERROR.Fatalln("Failed to load mime type config:", err)
	}
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

// Equal is a helper for comparing value equality, following these rules:
//  - Values with equivalent types are compared with reflect.DeepEqual
//  - int, uint, and float values are compared without regard to the type width.
//    for example, Equal(int32(5), int64(5)) == true
//  - strings and byte slices are converted to strings before comparison.
//  - else, return false.
func Equal(a, b interface{}) bool {
	if reflect.TypeOf(a) == reflect.TypeOf(b) {
		return reflect.DeepEqual(a, b)
	}
	switch a.(type) {
	case int, int8, int16, int32, int64:
		switch b.(type) {
		case int, int8, int16, int32, int64:
			return reflect.ValueOf(a).Int() == reflect.ValueOf(b).Int()
		}
	case uint, uint8, uint16, uint32, uint64:
		switch b.(type) {
		case uint, uint8, uint16, uint32, uint64:
			return reflect.ValueOf(a).Uint() == reflect.ValueOf(b).Uint()
		}
	case float32, float64:
		switch b.(type) {
		case float32, float64:
			return reflect.ValueOf(a).Float() == reflect.ValueOf(b).Float()
		}
	case string:
		switch b.(type) {
		case []byte:
			return a.(string) == string(b.([]byte))
		}
	case []byte:
		switch b.(type) {
		case string:
			return b.(string) == string(a.([]byte))
		}
	}
	return false
}

// ClientIP method returns client IP address from HTTP request.
//
// Note: Set property "app.behind.proxy" to true only if Revel is running
// behind proxy like nginx, haproxy, apache, etc. Otherwise
// you may get inaccurate Client IP address. Revel parses the
// IP address in the order of X-Forwarded-For, X-Real-IP.
//
// By default revel will get http.Request's RemoteAddr
func ClientIP(r *http.Request) string {
	if Config.BoolDefault("app.behind.proxy", false) {
		// Header X-Forwarded-For
		if fwdFor := strings.TrimSpace(r.Header.Get(hdrForwardedFor)); fwdFor != "" {
			index := strings.Index(fwdFor, ",")
			if index == -1 {
				return fwdFor
			}
			return fwdFor[:index]
		}

		// Header X-Real-Ip
		if realIP := strings.TrimSpace(r.Header.Get(hdrRealIP)); realIP != "" {
			return realIP
		}
	}

	if remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return remoteAddr
	}

	return ""
}

// createDir method creates nested directories if not exists
func createDir(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("Failed to create directory '%v': %v", path, err)
			}
		} else {
			return fmt.Errorf("Failed to create directory '%v': %v", path, err)
		}
	}
	return nil
}

func init() {
	OnAppStart(LoadMimeConfig)
}
