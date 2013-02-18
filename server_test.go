package revel

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
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

func (c Hotels) Show(id int) Result {
	title := "View Hotel"
	hotel := &Hotel{id, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300}
	return c.Render(title, hotel)
}

// This tries to benchmark the usual request-serving pipeline to get an overall
// performance metric.
//
// Each iteration runs one mock request to display a hotel's detail page by id,
// and 10 static requests for a file, sized 7k.
//
// Contributing parts:
// - Routing
// - Controller lookup / invocation
// - Parameter binding
// - Session, flash, i18n cookies
// - Render() call magic
// - Validation magic
// - Template rendering
func BenchmarkRevel(b *testing.B) {
	Init("", "github.com/robfig/revel/samples/booking", "")

	TRACE = log.New(ioutil.Discard, "", 0)
	INFO = TRACE
	WARN = TRACE
	ERROR = TRACE

	MainRouter = NewRouter("")
	routesFile, _ := ioutil.ReadFile(filepath.Join(BasePath, "conf", "routes"))
	MainRouter.parse(string(routesFile), false)
	MainTemplateLoader = NewTemplateLoader([]string{ViewsPath})
	MainTemplateLoader.Refresh()

	RegisterController((*Hotels)(nil),
		[]*MethodType{
			&MethodType{
				Name: "Show",
				Args: []*MethodArg{
					{"id", reflect.TypeOf((*int)(nil))},
				},
				RenderArgNames: map[int][]string{30: []string{"title", "hotel"}},
			},
		})

	plugins.OnAppStart()

	showRequest, _ := http.NewRequest("GET", "/hotels/3", nil)
	staticRequest, _ := http.NewRequest("GET", "/public/js/sessvars.js", nil)

	resp := httptest.NewRecorder()

	// First, test that the expected responses are actually generated
	handle(resp, showRequest)
	if !strings.Contains(resp.Body.String(), "300 Main St.") {
		b.Errorf("Failed to find hotel address in action response:\n%s", resp.Body)
		b.FailNow()
	}
	resp.Body.Reset()

	handle(resp, staticRequest)
	if resp.Body.Len() != 6712 {
		b.Errorf("Expected sessvars.js to have 6712 bytes, got %d:\n%s", resp.Body.Len(), resp.Body)
		b.FailNow()
	}

	resp.Body = nil
	b.ResetTimer()

	// Each iteration, run 1 action and 10 static requests.
	for i := 0; i < b.N; i++ {
		handle(resp, showRequest)
		for j := 0; j < 10; j++ {
			handle(resp, staticRequest)
		}
	}
}
