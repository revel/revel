package rev

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
)

type Route struct {
	method string // e.g. GET
	path   string // e.g. /app/{id}
	action string // e.g. Application.ShowApp

	pathPattern   *regexp.Regexp // for matching the url path
	staticDir     string         // e.g. "public" from action "staticDir:public"
	args          []*arg         // e.g. {id} from path /app/{id}
	actionArgs    []string
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

// TODO: Use exp/regexp and named groups e.g. (?P<name>a)
var nakedPathParamRegex *regexp.Regexp = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z_0-9]*)\}`)
var argsPattern *regexp.Regexp = regexp.MustCompile(`\{<(?P<pattern>[^>]+)>(?P<var>[a-zA-Z_0-9]+)\}`)

// Prepares the route to be used in matching.
func NewRoute(method, path, action string) (r *Route) {
	r = &Route{
		method: strings.ToUpper(method),
		path:   path,
		action: action,
	}

	// Handle static routes
	if strings.HasPrefix(r.action, "staticDir:") {
		if r.method != "*" && r.method != "GET" {
			LOG.Print("W: Static route only supports GET")
			return
		}

		if !strings.HasSuffix(r.path, "/") {
			LOG.Printf("W: The path for staticDir must end with / (%s)", r.path)
			r.path = r.path + "/"
		}

		r.pathPattern = regexp.MustCompile("^" + r.path + "(.*)$")
		r.staticDir = r.action[len("staticDir:"):]
		// TODO: staticFile:
		return
	}

	// URL pattern
	// TODO: Support non-absolute paths
	if !strings.HasPrefix(r.path, "/") {
		LOG.Print("E: Absolute URL required.")
		return
	}

	// Handle embedded arguments

	// Convert path arguments with unspecified regexes to standard form.
	// e.g. "/customer/{id}" => "/customer/{<[^/]+>id}
	normPath := nakedPathParamRegex.ReplaceAllStringFunc(r.path, func(m string) string {
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
	var actionPatternStr string = strings.Replace(r.action, ".", `\.`, -1)
	for _, arg := range r.args {
		var argName string = "{" + arg.name + "}"
		if argIndex := strings.Index(actionPatternStr, argName); argIndex != -1 {
			actionPatternStr = strings.Replace(actionPatternStr, argName,
				"(?P<"+arg.name+">"+arg.constraint.String()+")", -1)
			r.actionArgs = append(r.actionArgs, arg.name)
		}
	}
	r.actionPattern = regexp.MustCompile(actionPatternStr)
	return
}

// Return nil if no match.
func (r *Route) Match(method string, reqPath string) *RouteMatch {
	// Check the Method
	if r.method != "*" && method != r.method && !(method == "HEAD" && r.method == "GET") {
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
	action := r.action
	if strings.Contains(action, "{") {
		for key, value := range params {
			action = strings.Replace(action, "{"+key+"}", value, -1)
		}
	}

	// Split the action into controller and method
	actionSplit := strings.Split(action, ".")
	if len(actionSplit) != 2 {
		LOG.Printf("E: Failed to split action: %s", r.action)
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
	routes []*Route
}

func (router *Router) Route(req *http.Request) *RouteMatch {
	for _, route := range router.routes {
		if m := route.Match(req.Method, req.URL.Path); m != nil {
			return m
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

// Load the routes file.
func LoadRoutes() *Router {
	// Get the routes file content.
	contentBytes, err := ioutil.ReadFile(path.Join(BasePath, "conf", "routes"))
	if err != nil {
		LOG.Fatalln("Failed to load routes file:", err)
	}
	content := string(contentBytes)
	return NewRouter(content)
}

func parseRouteLine(line string) (method, path, action string, found bool) {
	var matches []string = routePattern.FindStringSubmatch(line)
	if matches == nil {
		return
	}
	method, path, action = matches[1], matches[4], matches[5]
	found = true
	return
}

func NewRouter(routesConf string) *Router {
	router := new(Router)
	routes := make([]*Route, 0, 10)

	// For each line..
	for _, line := range strings.Split(routesConf, "\n") {
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
	}

	router.routes = routes
	return router
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
	for _, route := range router.routes {
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
		path := route.path
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

		method := route.method
		star := false
		if route.method == "*" {
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
	LOG.Println("Failed to find reverse route:", action, argValues)
	return nil
}
