package play

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

var router *Router
var templateLoader *TemplateLoader

// This method handles all http requests.
func handle(w http.ResponseWriter, r *http.Request) {
	// Figure out the Controller/Action
	var route *RouteMatch = router.Route(r)
	if route == nil {
		http.NotFound(w, r)
		return
	}

	// Dispatch the static files first.
	if route.StaticFilename != "" {
		http.ServeFile(w, r, route.StaticFilename)
		return
	}

	// Invoke the controller method...
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
	var method reflect.Value = appControllerPtr.MethodByName(route.FunctionName)
	if !method.IsValid() {
		LOG.Printf("E: Function %s not found on Controller %s",
			route.FunctionName, route.ControllerName)
		http.NotFound(w, r)
		return
	}

	// Get the types of the method parameters.
	var methodType reflect.Type = method.Type()
	var paramTypes []reflect.Type = make([]reflect.Type, 0, 5)
	for i := 0; i < methodType.NumIn(); i++ {
		paramTypes = append(paramTypes, methodType.In(i))
	}

	// Check they are equal.
	if len(paramTypes) != len(route.Params) {
		LOG.Printf("E: # matched params (%d) != # expected params (%d)",
			len(route.Params), len(paramTypes))
		http.NotFound(w, r)
		return
	}

	// Get the values of the method params
	paramValues := make([]reflect.Value, 0, 5)
	for i := 0; i < len(route.Params); i++ {
		// For each, parse it into the type that the method expects.
		var param string = route.Params[i]
		var paramType reflect.Type = paramTypes[i]
		var paramValue reflect.Value
		switch (paramType.Kind()) {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intParam, err := strconv.Atoi(param)
			if err != nil {
				LOG.Printf("Failed to parse param to int: %s", param)
				http.NotFound(w, r)
				return
			}
			paramValue = reflect.ValueOf(intParam)
		}
		paramValues = append(paramValues, paramValue)
	}

	// Call the method.
	resultValue := method.Call(paramValues)[0]
	result := resultValue.Interface().(*Result)
	w.Write([]byte(result.body))
}

// Run the server.
// This is called from the generated main file.
func Run(port int) {
	// Load the routes
	router = LoadRoutes()

	templateLoader = new(TemplateLoader)

	// Now that we know all the Controllers, start the server.
	LOG.Printf("Listening on port %d...", port)
	http.HandleFunc("/", handle)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		LOG.Fatalln("Failed to listen:", err)
	}
}


