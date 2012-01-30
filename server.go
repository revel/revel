package play

import (
	"fmt"
	"net/http"
	"reflect"
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
	LOG.Printf("Calling %s.%s", route.ControllerName, route.MethodName)
	var controllerType *ControllerType = LookupControllerType(route.ControllerName)
	if controllerType == nil {
		LOG.Printf("E: Controller %s not found", route.ControllerName)
		http.NotFound(w, r)
		return
	}

	// Create an AppController.
	var appControllerPtr reflect.Value = reflect.New(controllerType.Type)
	var appController reflect.Value = appControllerPtr.Elem()

	// Create and configure Play Controller
	var c *Controller = &Controller{
		request: r,
		responseWriter: w,
		name: controllerType.Type.Name(),
		controllerType: controllerType,
	}

	// Set the embedded Play Controller field, in the App Controller
	var controllerField reflect.Value = appController.Field(0)
	controllerField.Set(reflect.ValueOf(c))

	// Now call the action.
	methodType := controllerType.Method(route.MethodName)
	if methodType == nil {
		LOG.Println("E: Failed to find method", route.MethodName, "on Controller",
			route.ControllerName)
		http.NotFound(w, r)
		return
	}

	var method reflect.Value = appControllerPtr.MethodByName(route.MethodName)
	if !method.IsValid() {
		LOG.Printf("E: Function %s not found on Controller %s",
			route.MethodName, route.ControllerName)
		http.NotFound(w, r)
		return
	}

	// Collect the values for the method's arguments.
	var actualArgs []reflect.Value
	for _, arg := range methodType.Args {
		// If this arg is provided, add it to actualArgs
		// Else, leave it as the default 0 value.
		if value, ok := route.Params[arg.Name]; ok {
			actualArgs = append(actualArgs, Bind(arg.Type, value))
		} else {
			actualArgs = append(actualArgs, reflect.Zero(arg.Type))
		}
	}

	// Call the method.
	resultValue := method.Call(actualArgs)[0]
	result := resultValue.Interface().(*Result)
	w.Write([]byte(result.body))
}

// Run the server.
// This is called from the generated main file.
func Run(port int) {
	// Load the routes
	// TODO: Watch the routes file for changes, and reload.
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


