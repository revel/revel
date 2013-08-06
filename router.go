package revel

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang/glog"
)

type Route struct {
	Method      string   // e.g. GET
	Path        string   // e.g. /app/{id}
	Action      string   // e.g. Application.ShowApp
	FixedParams []string // e.g. "arg1","arg2","arg3" (CSV formatting)

	pathPattern   *regexp.Regexp // for matching the url path
	args          []*arg         // e.g. {id} from path /app/{id}
	actionPattern *regexp.Regexp
}

type RouteMatch struct {
	Action         string // e.g. Application.ShowApp
	ControllerName string // e.g. Application
	MethodName     string // e.g. ShowApp
	FixedParams    []string
	Params         map[string]string // e.g. {id: 123}
}

type arg struct {
	name       string
	index      int
	constraint *regexp.Regexp
}

var (
	nakedPathParamRegex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z_0-9]*)\}`)
	argsPattern         = regexp.MustCompile(`\{<(?P<pattern>[^>]+)>(?P<var>[a-zA-Z_0-9]+)\}`)
)

// Prepares the route to be used in matching.
func NewRoute(method, path, action, fixedArgs string) (r *Route) {
	// Handle fixed arguments
	argsReader := strings.NewReader(fixedArgs)
	csv := csv.NewReader(argsReader)
	fargs, err := csv.Read()
	if err != nil && err != io.EOF {
		glog.Errorf("Invalid fixed parameters (%v): for string '%v'", err.Error(), fixedArgs)
	}

	r = &Route{
		Method:      strings.ToUpper(method),
		Path:        path,
		Action:      action,
		FixedParams: fargs,
	}

	// URL pattern
	// TODO: Support non-absolute paths
	if !strings.HasPrefix(r.Path, "/") {
		glog.Error("Absolute URL required.")
		return
	}

	// Handle embedded arguments

	// Convert path arguments with unspecified regexes to standard form.
	// e.g. "/customer/{id}" => "/customer/{<[^/]+>id}
	normPath := nakedPathParamRegex.ReplaceAllStringFunc(r.Path, func(m string) string {
		var argMatches []string = nakedPathParamRegex.FindStringSubmatch(m)
		return "{<[^/]+>" + argMatches[1] + "}"
	})

	// Go through the arguments
	r.args = make([]*arg, 0, 3)
	for i, m := range argsPattern.FindAllStringSubmatch(normPath, -1) {
		r.args = append(r.args, &arg{
			name:       string(m[2]),
			index:      i,
			constraint: regexp.MustCompile(string(m[1])),
		})
	}

	// Now assemble the entire path regex, including the embedded parameters.
	// e.g. /app/{<[^/]+>id} => /app/(?P<id>[^/]+)
	pathPatternStr := argsPattern.ReplaceAllStringFunc(normPath, func(m string) string {
		var argMatches []string = argsPattern.FindStringSubmatch(m)
		return "(?P<" + argMatches[2] + ">" + argMatches[1] + ")"
	})
	r.pathPattern = regexp.MustCompile(pathPatternStr + "$")

	// Handle action
	var actionPatternStr string = strings.Replace(r.Action, ".", `\.`, -1)
	for _, arg := range r.args {
		var argName string = "{" + arg.name + "}"
		if argIndex := strings.Index(actionPatternStr, argName); argIndex != -1 {
			actionPatternStr = strings.Replace(actionPatternStr, argName,
				"(?P<"+arg.name+">"+arg.constraint.String()+")", -1)
		}
	}
	r.actionPattern = regexp.MustCompile(actionPatternStr)
	return
}

// Return nil if no match.
func (r *Route) Match(method string, reqPath string) *RouteMatch {
	// Check the Method
	if r.Method != "*" && method != r.Method && !(method == "HEAD" && r.Method == "GET") {
		return nil
	}

	// Check the Path
	var matches []string = r.pathPattern.FindStringSubmatch(reqPath)
	if len(matches) == 0 || len(matches[0]) != len(reqPath) {
		return nil
	}

	// Figure out the Param names.
	params := make(map[string]string)
	for i, m := range matches[1:] {
		params[r.pathPattern.SubexpNames()[i+1]] = m
	}

	// If the action is variablized, replace into it with the captured args.
	action := r.Action
	if strings.Contains(action, "{") {
		for key, value := range params {
			action = strings.Replace(action, "{"+key+"}", value, -1)
		}
	}

	// Special handling for explicit 404's.
	if action == "404" {
		return &RouteMatch{
			Action: "404",
		}
	}

	// Split the action into controller and method
	actionSplit := strings.Split(action, ".")
	if len(actionSplit) != 2 {
		glog.Errorf("Failed to split action: %s (matching route: %s)", action, r.Action)
		return nil
	}

	return &RouteMatch{
		Action:         action,
		ControllerName: actionSplit[0],
		MethodName:     actionSplit[1],
		Params:         params,
		FixedParams:    r.FixedParams,
	}
}

type Router struct {
	Routes []*Route
	path   string
}

func (router *Router) Route(req *http.Request) *RouteMatch {
	for _, route := range router.Routes {
		if m := route.Match(req.Method, req.URL.Path); m != nil {
			return m
		}
	}
	return nil
}

// Refresh re-reads the routes file and re-calculates the routing table.
// Returns an error if a specified action could not be found.
func (router *Router) Refresh() (err *Error) {
	router.Routes, err = parseRoutesFile(router.path, true)
	return
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

		route := NewRoute(method, path, action, fixedArgs)
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
	// Skip variable routes.
	if strings.ContainsAny(route.Action, "{}") {
		return nil
	}

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
		glog.Infoln("Skipping routes for inactive module", moduleName)
		return nil, nil
	}
	return parseRoutesFile(filepath.Join(module.Path, "conf", "routes"), validate)
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

NEXT_ROUTE:
	// Loop through the routes.
	for _, route := range router.Routes {
		if route.actionPattern == nil {
			continue
		}

		var matches []string = route.actionPattern.FindStringSubmatch(action)
		if len(matches) == 0 {
			continue
		}

		for i, match := range matches[1:] {
			argValues[route.actionPattern.SubexpNames()[i+1]] = match
		}

		// Create a lookup for the route args.
		routeArgs := make(map[string]*arg)
		for _, arg := range route.args {
			routeArgs[arg.name] = arg
		}

		// Enforce the constraints on the arg values.
		for argKey, argValue := range argValues {
			arg, ok := routeArgs[argKey]
			if ok && !arg.constraint.MatchString(argValue) {
				continue NEXT_ROUTE
			}
		}

		// Build up the URL.
		var queryValues url.Values = make(url.Values)
		// Handle optional trailing slashes (e.g. "/?") by removing the question mark.
		path := strings.Replace(route.Path, "?", "", -1)
		for argKey, argValue := range argValues {
			if _, ok := routeArgs[argKey]; ok {
				// If this arg goes into the path, put it in.
				path = regexp.MustCompile(`\{(<[^>]+>)?`+regexp.QuoteMeta(argKey)+`\}`).
					ReplaceAllString(path, url.QueryEscape(string(argValue)))
			} else {
				// Else, add it to the query string.
				queryValues.Set(argKey, argValue)
			}
		}

		// Calculate the final URL and Method
		url := path
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
	glog.Errorln("Failed to find reverse route:", action, argValues)
	return nil
}

func init() {
	OnAppStart(func() {
		MainRouter = NewRouter(filepath.Join(BasePath, "conf", "routes"))
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
	for k, v := range route.Params {
		if c.Params.Route == nil {
			c.Params.Route = make(map[string][]string)
		}
		c.Params.Route[k] = []string{v}
	}

	// Add the fixed parameters mapped by name.
	for i, value := range route.FixedParams {
		if c.Params.Fixed == nil {
			c.Params.Fixed = make(url.Values)
		}
		if i < len(c.MethodType.Args) {
			arg := c.MethodType.Args[i]
			c.Params.Fixed.Set(arg.Name, value)
		} else {
			glog.Warningln("Too many parameters to", route.Action, "trying to add", value)
			break
		}
	}

	fc[0](c, fc[1:])
}
