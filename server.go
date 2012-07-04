package rev

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

var (
	MainRouter         *Router
	MainTemplateLoader *TemplateLoader

	websocketType = reflect.TypeOf((*websocket.Conn)(nil))
)

// This method handles all requests.  It dispatches to handleInternal after
// handling / adapting websocket connections.
func handle(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") == "websocket" {
		websocket.Handler(func(ws *websocket.Conn) {
			r.Method = "WS"
			handleInternal(w, r, ws)
		}).ServeHTTP(w, r)
	} else {
		handleInternal(w, r, nil)
	}
}

func handleInternal(w http.ResponseWriter, r *http.Request, ws *websocket.Conn) {
	// TODO: StaticPathsCache
	req, resp := NewRequest(r), NewResponse(w)

	// Figure out the Controller/Action
	var route *RouteMatch = MainRouter.Route(r)
	if route == nil {
		NotFound(req, resp, "No matching route found")
		return
	}

	// Dispatch the static files first.
	if route.StaticFilename != "" {
		http.ServeFile(w, r, route.StaticFilename)
		return
	}

	// Construct the controller and get the method to call.
	controller, appControllerPtr := NewAppController(req, resp, route.ControllerName, route.MethodName)
	if controller == nil {
		NotFound(req, resp, fmt.Sprintln("No matching action found:", route.Action))
		return
	}

	var method reflect.Value = appControllerPtr.MethodByName(controller.MethodType.Name)
	if !method.IsValid() {
		LOG.Printf("E: Function %s not found on Controller %s",
			route.MethodName, route.ControllerName)
		NotFound(req, resp, fmt.Sprintln("No matching action found:", route.Action))
		return
	}

	// Add the route Params to the Request Params.
	for key, value := range route.Params {
		url.Values(controller.Params.Values).Add(key, value)
	}

	// Collect the values for the method's arguments.
	var actualArgs []reflect.Value
	for _, arg := range controller.MethodType.Args {
		// If they accept a websocket connection, treat that arg specially.
		var boundArg reflect.Value
		if arg.Type == websocketType {
			boundArg = reflect.ValueOf(ws)
		} else {
			boundArg = controller.Params.Bind(arg.Name, arg.Type)
		}
		actualArgs = append(actualArgs, boundArg)
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
	MainRouter = LoadRoutes()
	MainTemplateLoader = NewTemplateLoader(ViewsPath, RevelTemplatePath)

	go func() {
		time.Sleep(100 * time.Millisecond)
		LOG.Printf("Listening on port %d...", port)
	}()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(handle),
	}

	plugins.OnAppStart()

	err := server.ListenAndServe()
	if err != nil {
		LOG.Fatalln("Failed to listen:", err)
	}
}
