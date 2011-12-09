package play

import (
	"net/http"
	// "path/filepath"
	// "go/token"
	// "go/parser"
	// "go/ast"
	// "io/ioutil"
	"log"
	"reflect"
)

var router *Router

func handle(w http.ResponseWriter, r *http.Request) {
	// Figure out the Controller/Action
	var route *RouteMatch = router.Route(r)
	if route == nil {
		http.NotFound(w, r)
		return
	}

	LOG.Printf("Calling %s.%s", route.ControllerName, route.FunctionName)
	var t reflect.Type = LookupControllerType(route.ControllerName)

	// Create an AppController.
	var appControllerPtr reflect.Value = reflect.New(t)
	var appController reflect.Value = appControllerPtr.Elem()

	// Create and configure Play Controller
	var c *Controller = &Controller{
		request: r,
		responseWriter: w,
		name: t.Name(),
	}

	// Set the embedded Play Controller field, in the App Controller
	var controllerField reflect.Value = appController.Field(0)
	controllerField.Set(reflect.ValueOf(c))

	// Now call the action.
	// TODO: Figure out the arguments it expects, and try to bind parameters to
	// them.
	method := appControllerPtr.MethodByName(route.FunctionName)
	if !method.IsValid() {
		LOG.Printf("E: Function %s not found on Controller %s",
			route.FunctionName, route.ControllerName)
		http.NotFound(w, r)
		return
	}

	resultValue := method.Call([]reflect.Value{ })[0]
	result := resultValue.Interface().(*Result)
	w.Write([]byte(result.body))
}

func Run() {
	// TODO: Scan the application directory and automatically register Controllers
	// fset := token.NewFileSet()
	// fileInfos, _ := ioutil.ReadDir(filepath.Join(AppPath, "controllers"))
	// for i, file := range(fileInfos) {
	// 	fset.AddFile(file.Name(), fset.Base(), int(file.Size()))
	// }
	// var a map[string]*ast.Package
	// a , _ = parser.ParseDir(fset, AppPath, nil, 0)

	// Load the routes
	router = LoadRoutes()

	// Now that we know all the Controllers, start the server.
	log.Printf("Listening on port 8080...")
	http.HandleFunc("/", handle)
	http.ListenAndServe(":8080", nil)
}
