package play

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

var router *Router
var templateLoader *TemplateLoader

// This method handles all http requests.
func handle(w http.ResponseWriter, r *http.Request) {
	// TODO: StaticPathsCache

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

	// Construct the controller and get the method to call.
	controller, appControllerPtr := NewAppController(w, r, route.ControllerName, route.MethodName)
	if controller == nil {
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

	// Add the route Params to the Request Params.
	for key, value := range route.Params {
		url.Values(controller.Params).Add(key, value)
	}

	// Collect the values for the method's arguments.
	var actualArgs []reflect.Value
	for _, arg := range controller.MethodType.Args {
		actualArgs = append(actualArgs, controller.Params.Bind(arg.Type, arg.Name))
	}

	// Invoke the method.
	// (Note that the method Value is already bound to the appController receiver.)
	controller.Invoke(appControllerPtr, method, actualArgs)
}

// Run the server.
// This is called from the generated main file.
func Run(port int) {
	// Load the routes
	// TODO: Watch the routes file for changes, and reload.
	router = LoadRoutes()
	templateLoader = NewTemplateLoader()

	// Now that we know all the Controllers, start the server.
	go func() {
		time.Sleep(100 * time.Millisecond)
		LOG.Printf("Listening on port %d...", port)
	}()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(handle),
	}

	err := server.ListenAndServe()
	if err != nil {
		LOG.Fatalln("Failed to listen:", err)
	}
}
