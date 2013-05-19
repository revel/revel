package revel

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// A copy of the hotel struct from the sample app.
type Hotel struct {
	HotelId          int
	Name, Address    string
	City, State, Zip string
	Country          string
	Price            int
}

type Hotels struct {
	*Controller
}

type Static struct {
	*Controller
}

func (c Hotels) Show(id int) Result {
	title := "View Hotel"
	hotel := &Hotel{id, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300}
	return c.Render(title, hotel)
}

func (c Hotels) Book(id int) Result {
	hotel := &Hotel{id, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300}
	return c.RenderJson(hotel)
}

func (c Static) Serve(prefix, filepath string) Result {
	var basePath, dirName string

	if !path.IsAbs(dirName) {
		basePath = BasePath
	}

	fname := path.Join(basePath, prefix, filepath)
	file, err := os.Open(fname)
	if os.IsNotExist(err) {
		return c.NotFound("")
	} else if err != nil {
		WARN.Printf("Problem opening file (%s): %s ", fname, err)
		return c.NotFound("This was found but not sure why we couldn't open it.")
	}
	return c.RenderFile(file, "")
}

// This tries to benchmark the usual request-serving pipeline to get an overall
// performance metric.
//
// Each iteration runs one mock request to display a hotel's detail page by id.
//
// Contributing parts:
// - Routing
// - Controller lookup / invocation
// - Parameter binding
// - Session, flash, i18n cookies
// - Render() call magic
// - Template rendering
func BenchmarkServeAction(b *testing.B) {
	benchmarkRequest(b, showRequest)
}

func BenchmarkServeJson(b *testing.B) {
	benchmarkRequest(b, jsonRequest)
}

// This tries to benchmark the static serving overhead when serving an "average
// size" 7k file.
func BenchmarkServeStatic(b *testing.B) {
	benchmarkRequest(b, staticRequest)
}

func benchmarkRequest(b *testing.B, req *http.Request) {
	startFakeBookingApp()
	b.ResetTimer()
	resp := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		handle(resp, req)
	}
}

// Test that the booking app can be successfully run for a test.
func TestFakeServer(t *testing.T) {
	startFakeBookingApp()

	resp := httptest.NewRecorder()

	// First, test that the expected responses are actually generated
	handle(resp, showRequest)
	if !strings.Contains(resp.Body.String(), "300 Main St.") {
		t.Errorf("Failed to find hotel address in action response:\n%s", resp.Body)
		t.FailNow()
	}
	resp.Body.Reset()

	handle(resp, staticRequest)
	if resp.Body.Len() != 6712 {
		t.Errorf("Expected sessvars.js to have 6712 bytes, got %d:\n%s", resp.Body.Len(), resp.Body)
		t.FailNow()
	}
	resp.Body.Reset()

	handle(resp, jsonRequest)
	if !strings.Contains(resp.Body.String(), `"Address":"300 Main St."`) {
		t.Errorf("Failed to find hotel address in JSON response:\n%s", resp.Body)
		t.FailNow()
	}

	resp.Body = nil
}

var (
	showRequest, _   = http.NewRequest("GET", "/hotels/3", nil)
	staticRequest, _ = http.NewRequest("GET", "/public/js/sessvars.js", nil)
	jsonRequest, _   = http.NewRequest("GET", "/hotels/3/booking", nil)
)

func startFakeBookingApp() {
	Init("", "github.com/robfig/revel/samples/booking", "")

	// Disable logging.
	TRACE = log.New(ioutil.Discard, "", 0)
	INFO = TRACE
	WARN = TRACE
	ERROR = TRACE

	MainRouter = NewRouter("")
	routesFile, _ := ioutil.ReadFile(filepath.Join(BasePath, "conf", "routes"))
	MainRouter.Routes, _ = parseRoutes("", string(routesFile), false)
	MainTemplateLoader = NewTemplateLoader([]string{ViewsPath, path.Join(RevelPath, "templates")})
	MainTemplateLoader.Refresh()

	RegisterController((*Hotels)(nil),
		[]*MethodType{
			&MethodType{
				Name: "Index",
			},
			&MethodType{
				Name: "Show",
				Args: []*MethodArg{
					{"id", reflect.TypeOf((*int)(nil))},
				},
				RenderArgNames: map[int][]string{36: []string{"title", "hotel"}},
			},
			&MethodType{
				Name: "Book",
				Args: []*MethodArg{
					{"id", reflect.TypeOf((*int)(nil))},
				},
			},
		})

	RegisterController((*Static)(nil),
		[]*MethodType{
			&MethodType{
				Name: "Serve",
				Args: []*MethodArg{
					&MethodArg{Name: "prefix", Type: reflect.TypeOf((*string)(nil))},
					&MethodArg{Name: "filepath", Type: reflect.TypeOf((*string)(nil))},
				},
				RenderArgNames: map[int][]string{},
			},
		})

	plugins.OnAppStart()
}
