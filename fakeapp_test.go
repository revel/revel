package revel

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
)

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

func (c Hotels) Index() Result {
	return c.RenderText("Hello, World!")
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

func startFakeBookingApp() {
	Init("prod", "github.com/revel/revel/testdata", "")

	// Disable logging.
	TRACE = log.New(ioutil.Discard, "", 0)
	INFO = TRACE
	WARN = TRACE
	ERROR = TRACE

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
				RenderArgNames: map[int][]string{30: []string{"title", "hotel"}},
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

	runStartupHooks()
}
