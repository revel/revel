package play

import (
	"net/http"
	"log"
	"reflect"
	"runtime"
	"strings"
)

type Controller struct {
	request *http.Request
	responseWriter http.ResponseWriter
	name string
	controllerType *ControllerType
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

// Internal bookeeping

type ControllerType struct {
	Type reflect.Type
	Methods []*MethodType
}

type MethodType struct {
	Name string
	Args []*MethodArg
}

type MethodArg struct {
	Name string
	Type reflect.Type
}

func (ct *ControllerType) Method(name string) *MethodType {
	for _, method := range ct.Methods {
		if method.Name == name {
			return method
		}
	}
	return nil
}

var controllers map[string]*ControllerType = make(map[string]*ControllerType)

func RegisterController(c interface{}, methods []*MethodType) {
	var t reflect.Type = reflect.TypeOf(c)
	var elem reflect.Type = t.Elem()
	controllers[elem.Name()] = &ControllerType{Type: elem, Methods: methods}
	log.Printf("Registered controller: %s", elem.Name())
}

func LookupControllerType(name string) *ControllerType {
	return controllers[name]
}

