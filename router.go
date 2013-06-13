package revel

import (
	"encoding/csv"
	"fmt"
	"github.com/robfig/pathtree"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
)

type Route struct {
	Method         string   // e.g. GET
	Path           string   // e.g. /app/:id
	Action         string   // e.g. "Application.ShowApp", "404"
	ControllerName string   // e.g. "Application", ""
	MethodName     string   // e.g. "ShowApp", ""
	FixedParams    []string // e.g. "arg1","arg2","arg3" (CSV formatting)
	TreePath       string   // e.g. "/GET/app/:id"

	routesPath string // e.g. /Users/robfig/gocode/src/myapp/conf/routes
	line       int    // e.g. 3
}

type RouteMatch struct {
	Action         string // e.g. 404
	ControllerName string // e.g. Application
	MethodName     string // e.g. ShowApp
	FixedParams    []string
	Params         map[string][]string // e.g. {id: 123}
}

type arg struct {
	name       string
	index      int
	constraint *regexp.Regexp
}

// Prepares the route to be used in matching.
func NewRoute(method, path, action, fixedArgs, routesPath string, line int) (r *Route) {
	// Handle fixed arguments
	argsReader := strings.NewReader(fixedArgs)
	csv := csv.NewReader(argsReader)
	fargs, err := csv.Read()
	if err != nil && err != io.EOF {
		ERROR.Printf("Invalid fixed parameters (%v): for string '%v'", err.Error(), fixedArgs)
	}

	r = &Route{
		Method:      strings.ToUpper(method),
		Path:        path,
		Action:      action,
		FixedParams: fargs,
		TreePath:    treePath(strings.ToUpper(method), path),
		routesPath:  routesPath,
		line:        line,
	}

	// URL pattern
	if !strings.HasPrefix(r.Path, "/") {
		ERROR.Print("Absolute URL required.")
		return
	}

	actionSplit := strings.Split(action, ".")
	if len(actionSplit) == 2 {
		r.ControllerName = actionSplit[0]
		r.MethodName = actionSplit[1]
	}

	return
}

func treePath(method, path string) string {
	if method == "*" {
		method = ":METHOD"
	}
	return "/" + method + path
}

type Router struct {
	Routes []*Route
	Tree   *pathtree.Node
	path   string // path to the routes file
}

var notFound = &RouteMatch{Action: "404"}

func (router *Router) Route(req *http.Request) *RouteMatch {
	leaf, expansions := router.Tree.Find(treePath(req.Method, req.URL.Path))
	if leaf == nil {
		return nil
	}
	route := leaf.Value.(*Route)

	// Create a map of the route parameters.
	var params url.Values
	if len(expansions) > 0 {
		params = make(url.Values)
		for i, v := range expansions {
			params[leaf.Wildcards[i]] = []string{v}
		}
	}

	// Special handling for explicit 404's.
	if route.Action == "404" {
		return notFound
	}

	// If the action is variablized, replace into it with the captured args.
	controllerName, methodName := route.ControllerName, route.MethodName
	if controllerName[0] == ':' {
		controllerName = params[controllerName[1:]][0]
	}
	if methodName[0] == ':' {
		methodName = params[methodName[1:]][0]
	}

	return &RouteMatch{
		ControllerName: controllerName,
		MethodName:     methodName,
		Params:         params,
		FixedParams:    route.FixedParams,
	}
}

// Refresh re-reads the routes file and re-calculates the routing table.
// Returns an error if a specified action could not be found.
func (router *Router) Refresh() (err *Error) {
	router.Routes, err = parseRoutesFile(router.path, true)
	if err != nil {
		return
	}
	err = router.updateTree()
	return
}

func (router *Router) updateTree() *Error {
	router.Tree = pathtree.New()
	for _, route := range router.Routes {
		err := router.Tree.Add(route.TreePath, route)

		// Allow GETs to respond to HEAD requests.
		if err == nil && route.Method == "GET" {
			err = router.Tree.Add(treePath("HEAD", route.Path), route)
		}

		// Error adding a route to the pathtree.
		if err != nil {
			return routeError(err, route.routesPath, "", route.line)
		}
	}
	return nil
}

// parseRoutesFile reads the given routes file and returns the contained routes.
func parseRoutesFile(routesPath string, validate bool) ([]*Route, *Error) {
	contentBytes, err := ioutil.ReadFile(routesPath)
	if err != nil {
		return nil, &Error{
			Title:       "Failed to load routes file",
			Description: err.Error(),
		}
	}
	return parseRoutes(routesPath, string(contentBytes), validate)
}

// parseRoutes reads the content of a routes file into the routing table.
func parseRoutes(routesPath, content string, validate bool) ([]*Route, *Error) {
	var routes []*Route

	// For each line..
	for n, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// Handle included routes from modules.
		// e.g. "module:testrunner" imports all routes from that module.
		if strings.HasPrefix(line, "module:") {
			moduleRoutes, err := getModuleRoutes(line[len("module:"):], validate)
			if err != nil {
				return nil, routeError(err, routesPath, content, n)
			}
			routes = append(routes, moduleRoutes...)
			continue
		}

		// A single route
		method, path, action, fixedArgs, found := parseRouteLine(line)
		if !found {
			continue
		}

		route := NewRoute(method, path, action, fixedArgs, routesPath, n)
		routes = append(routes, route)

		if validate {
			if err := validateRoute(route); err != nil {
				return nil, routeError(err, routesPath, content, n)
			}
		}
	}

	return routes, nil
}

// validateRoute checks that every specified action exists.
func validateRoute(route *Route) error {
	// Skip 404s
	if route.Action == "404" {
		return nil
	}

	// We should be able to load the action.
	parts := strings.Split(route.Action, ".")
	if len(parts) != 2 {
		return fmt.Errorf("Expected two parts (Controller.Action), but got %d: %s",
			len(parts), route.Action)
	}

	// Skip variable routes.
	if parts[0][0] == ':' || parts[1][0] == ':' {
		return nil
	}

	var c Controller
	if err := c.SetAction(parts[0], parts[1]); err != nil {
		return err
	}

	return nil
}

// routeError adds context to a simple error message.
func routeError(err error, routesPath, content string, n int) *Error {
	if revelError, ok := err.(*Error); ok {
		return revelError
	}
	// Load the route file content if necessary
	if content == "" {
		contentBytes, err := ioutil.ReadFile(routesPath)
		if err != nil {
			ERROR.Println("Failed to read route file %s: %s", routesPath, err)
		} else {
			content = string(contentBytes)
		}
	}
	return &Error{
		Title:       "Route validation error",
		Description: err.Error(),
		Path:        routesPath,
		Line:        n + 1,
		SourceLines: strings.Split(content, "\n"),
	}
}

// getModuleRoutes loads the routes file for the given module and returns the
// list of routes.
func getModuleRoutes(moduleName string, validate bool) ([]*Route, *Error) {
	// Look up the module.  It may be not found due to the common case of e.g. the
	// testrunner module being active only in dev mode.
	module, found := ModuleByName(moduleName)
	if !found {
		INFO.Println("Skipping routes for inactive module", moduleName)
		return nil, nil
	}
	return parseRoutesFile(path.Join(module.Path, "conf", "routes"), validate)
}

// Groups:
// 1: method
// 4: path
// 5: action
// 6: fixedargs
var routePattern *regexp.Regexp = regexp.MustCompile(
	"(?i)^(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD|WS|\\*)" +
		"[(]?([^)]*)(\\))?[ \t]+" +
		"(.*/[^ \t]*)[ \t]+([^ \t(]+)" +
		`\(?([^)]*)\)?[ \t]*$`)

func parseRouteLine(line string) (method, path, action, fixedArgs string, found bool) {
	var matches []string = routePattern.FindStringSubmatch(line)
	if matches == nil {
		return
	}
	method, path, action, fixedArgs = matches[1], matches[4], matches[5], matches[6]
	found = true
	return
}

func NewRouter(routesPath string) *Router {
	return &Router{
		Tree: pathtree.New(),
		path: routesPath,
	}
}

type ActionDefinition struct {
	Host, Method, Url, Action string
	Star                      bool
	Args                      map[string]string
}

func (a *ActionDefinition) String() string {
	return a.Url
}

func (router *Router) Reverse(action string, argValues map[string]string) *ActionDefinition {
	actionSplit := strings.Split(action, ".")
	if len(actionSplit) != 2 {
		ERROR.Print("revel/router: reverse router got invalid action ", action)
		return nil
	}
	controllerName, methodName := actionSplit[0], actionSplit[1]

	for _, route := range router.Routes {
		// Skip routes without either a ControllerName or MethodName
		if route.ControllerName == "" || route.MethodName == "" {
			continue
		}

		// Check that the action matches or is a wildcard.
		controllerWildcard := route.ControllerName[0] == ':'
		methodWildcard := route.MethodName[0] == ':'
		if (!controllerWildcard && route.ControllerName != controllerName) ||
			(!methodWildcard && route.MethodName != methodName) {
			continue
		}
		if controllerWildcard {
			argValues[route.ControllerName[1:]] = controllerName
		}
		if methodWildcard {
			argValues[route.MethodName[1:]] = methodName
		}

		// Build up the URL.
		var (
			queryValues  = make(url.Values)
			pathElements = strings.Split(route.Path, "/")
		)
		for i, el := range pathElements {
			if el == "" || el[0] != ':' {
				continue
			}

			val, ok := argValues[el[1:]]
			if !ok {
				val = "<nil>"
				ERROR.Print("revel/router: reverse route missing route arg ", el[1:])
			}
			pathElements[i] = val
			delete(argValues, el[1:])
			continue
		}

		// Add any args that were not inserted into the path into the query string.
		for k, v := range argValues {
			queryValues.Set(k, v)
		}

		// Calculate the final URL and Method
		url := strings.Join(pathElements, "/")
		if len(queryValues) > 0 {
			url += "?" + queryValues.Encode()
		}

		method := route.Method
		star := false
		if route.Method == "*" {
			method = "GET"
			star = true
		}

		return &ActionDefinition{
			Url:    url,
			Method: method,
			Star:   star,
			Action: action,
			Args:   argValues,
			Host:   "TODO",
		}
	}
	ERROR.Println("Failed to find reverse route:", action, argValues)
	return nil
}

func init() {
	OnAppStart(func() {
		MainRouter = NewRouter(path.Join(BasePath, "conf", "routes"))
		if MainWatcher != nil && Config.BoolDefault("watch.routes", true) {
			MainWatcher.Listen(MainRouter, MainRouter.path)
		} else {
			MainRouter.Refresh()
		}
	})
}

func RouterFilter(c *Controller, fc []Filter) {
	// Figure out the Controller/Action
	var route *RouteMatch = MainRouter.Route(c.Request.Request)
	if route == nil {
		c.Result = c.NotFound("No matching route found")
		return
	}

	// The route may want to explicitly return a 404.
	if route.Action == "404" {
		c.Result = c.NotFound("(intentionally)")
		return
	}

	// Set the action.
	if err := c.SetAction(route.ControllerName, route.MethodName); err != nil {
		c.Result = c.NotFound(err.Error())
		return
	}

	// Add the route and fixed params to the Request Params.
	c.Params.Route = route.Params

	// Add the fixed parameters mapped by name.
	// TODO: Pre-calculate this mapping.
	for i, value := range route.FixedParams {
		if c.Params.Fixed == nil {
			c.Params.Fixed = make(url.Values)
		}
		if i < len(c.MethodType.Args) {
			arg := c.MethodType.Args[i]
			c.Params.Fixed.Set(arg.Name, value)
		} else {
			WARN.Println("Too many parameters to", route.Action, "trying to add", value)
			break
		}
	}

	fc[0](c, fc[1:])
}
