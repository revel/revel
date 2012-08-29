package rev

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
)

type Route struct {
	Method string // e.g. GET
	Path   string // e.g. /app/{id}
	Action string // e.g. Application.ShowApp

	pathPattern   *regexp.Regexp // for matching the url path
	staticDir     string         // e.g. "public" from action "staticDir:public"
	args          []*arg         // e.g. {id} from path /app/{id}
	actionPattern *regexp.Regexp
}

type RouteMatch struct {
	Action         string            // e.g. Application.ShowApp
	ControllerName string            // e.g. Application
	MethodName     string            // e.g. ShowApp
	Params         map[string]string // e.g. {id: 123}
	StaticFilename string
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
func NewRoute(method, path, action string) (r *Route) {
	r = &Route{
		Method: strings.ToUpper(method),
		Path:   path,
		Action: action,
	}

	// Handle static routes
	if strings.HasPrefix(r.Action, "staticDir:") {
		if r.Method != "*" && r.Method != "GET" {
			WARN.Print("Static route only supports GET")
			return
		}

		if !strings.HasSuffix(r.Path, "/") {
			WARN.Printf("The path for staticDir must end with / (%s)", r.Path)
			r.Path = r.Path + "/"
		}

		r.pathPattern = regexp.MustCompile("^" + r.Path + "(.*)$")
		r.staticDir = r.Action[len("staticDir:"):]
		// TODO: staticFile:
		return
	}

	// URL pattern
	// TODO: Support non-absolute paths
	if !strings.HasPrefix(r.Path, "/") {
		ERROR.Print("Absolute URL required.")
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
	if matches == nil {
		return nil
	}

	// If it's a static file request..
	if r.staticDir != "" {
		return &RouteMatch{
			StaticFilename: path.Join(BasePath, r.staticDir, matches[1]),
		}
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
		ERROR.Printf("Failed to split action: %s", r.Action)
		return nil
	}

	return &RouteMatch{
		Action:         action,
		ControllerName: actionSplit[0],
		MethodName:     actionSplit[1],
		Params:         params,
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
func (router *Router) Refresh() *Error {
	// Get the routes file content.
	contentBytes, err := ioutil.ReadFile(router.path)
	if err != nil {
		return &Error{
			Title:       "Failed to load routes file",
			Description: err.Error(),
		}
	}

	return router.parse(string(contentBytes), true)
}

// parse takes the content of a routes file and turns it into the routing table.
func (router *Router) parse(content string, validate bool) *Error {
	routes := make([]*Route, 0, 10)

	// For each line..
	for n, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		method, path, action, found := parseRouteLine(line)
		if !found {
			continue
		}

		route := NewRoute(method, path, action)
		routes = append(routes, route)

		if validate {
			if err := router.validate(route); err != nil {
				err.Path = router.path
				err.Line = n + 1
				err.SourceLines = strings.Split(content, "\n")
				return err
			}
		}
	}

	router.Routes = routes
	return nil
}

// Check that every specified action exists.
func (router *Router) validate(route *Route) *Error {
	// Skip static routes
	if route.staticDir != "" {
		return nil
	}

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
		return &Error{
			Title: "Route validation error",
			Description: fmt.Sprintf("Expected two parts (Controller.Action), but got %d: %s",
				len(parts), route.Action),
		}
	}

	ct := LookupControllerType(parts[0])
	if ct == nil {
		return &Error{
			Title:       "Route validation error",
			Description: "Unrecognized controller: " + parts[0],
		}
	}

	mt := ct.Method(parts[1])
	if mt == nil {
		return &Error{
			Title:       "Route validation error",
			Description: "Unrecognized method: " + parts[1],
		}
	}
	return nil
}

// Groups:
// 1: method
// 4: path
// 5: action
var routePattern *regexp.Regexp = regexp.MustCompile(
	"(?i)^(GET|POST|PUT|DELETE|OPTIONS|HEAD|WS|\\*)" +
		"[(]?([^)]*)(\\))? +" +
		"(.*/[^ ]*) +([^ (]+)(.+)?( *)$")

func parseRouteLine(line string) (method, path, action string, found bool) {
	var matches []string = routePattern.FindStringSubmatch(line)
	if matches == nil {
		return
	}
	method, path, action = matches[1], matches[4], matches[5]
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
		path := route.Path
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
	ERROR.Println("Failed to find reverse route:", action, argValues)
	return nil
}
