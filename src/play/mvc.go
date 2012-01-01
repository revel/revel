package play

import (
	"net/http"
	//"io/ioutil"
	"log"
	//"path"
	"reflect"
	"runtime"
	//"os"
	"strings"
//	"html/template"
)

type Controller struct {
	request *http.Request
	responseWriter http.ResponseWriter
	name string
}

func (c *Controller) Render(arg interface{}) (*Result) {
	// Find the template.
	// Get the calling function name.
	pc, _, _, _ := runtime.Caller(1)
	// e.g. sample/app/controllers.(*Application).Index
	var fqViewName string = runtime.FuncForPC(pc).Name()
	var viewName string = fqViewName[
		strings.LastIndex(fqViewName, ".") + 1 : len(fqViewName)]

	// Refresh templates.
	err := templateLoader.LoadTemplates()
	if err != nil {
		c.responseWriter.Write([]byte(err.Html()))
		return &Result{}
	}

	// Render the template
	html, _ := templateLoader.RenderTemplate(c.name + "/" + viewName + ".html", arg)

	// Prepare the result
	r := new(Result)
	c.responseWriter.Write([]byte(html))
	return r
}

type Result struct {
	body string
}

// Need the home directory.


// Eventually the harness will run the Parser, check the AST for Controllers,
// and create a registration file.  For now, clients have to register:

var controllers map[string]reflect.Type = make(map[string]reflect.Type)

func RegisterController(c interface{}) {
	var t reflect.Type = reflect.TypeOf(c)
	var elem reflect.Type = t.Elem()
	controllers[elem.Name()] = elem
	log.Printf("Registered controller: %s", elem.Name())
}

func LookupControllerType(name string) reflect.Type {
	return controllers[name]
}
