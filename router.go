// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"os"
	"sync"

	"github.com/revel/pathtree"
	"github.com/revel/revel/logger"
)

const (
	httpStatusCode = "404"
)

type Route struct {
	ModuleSource        *Module         // Module name of route
	Method              string          // e.g. GET
	Path                string          // e.g. /app/:id
	Action              string          // e.g. "Application.ShowApp", "404"
	ControllerNamespace string          // e.g. "testmodule.",
	ControllerName      string          // e.g. "Application", ""
	MethodName          string          // e.g. "ShowApp", ""
	FixedParams         []string        // e.g. "arg1","arg2","arg3" (CSV formatting)
	TreePath            string          // e.g. "/GET/app/:id"
	TypeOfController    *ControllerType // The controller type (if route is not wild carded)

	routesPath string // e.g. /Users/robfig/gocode/src/myapp/conf/routes
	line       int    // e.g. 3
}

type RouteMatch struct {
	Action           string // e.g. 404
	ControllerName   string // e.g. Application
	MethodName       string // e.g. ShowApp
	FixedParams      []string
	Params           map[string][]string // e.g. {id: 123}
	TypeOfController *ControllerType     // The controller type
	ModuleSource     *Module             // The module
}

type ActionPathData struct {
	Key                 string            // The unique key
	ControllerNamespace string            // The controller namespace
	ControllerName      string            // The controller name
	MethodName          string            // The method name
	Action              string            // The action
	ModuleSource        *Module           // The module
	Route               *Route            // The route
	FixedParamsByName   map[string]string // The fixed parameters
	TypeOfController    *ControllerType   // The controller type
}

var (
	// Used to store decoded action path mappings
	actionPathCacheMap = map[string]*ActionPathData{}
	// Used to prevent concurrent writes to map
	actionPathCacheLock = sync.Mutex{}
	// The path returned if not found
	notFound = &RouteMatch{Action: "404"}
)

var routerLog = RevelLog.New("section", "router")

func init() {
	AddInitEventHandler(func(typeOf Event, value interface{}) (responseOf EventResponse) {
		// Add in an
		if typeOf == ROUTE_REFRESH_REQUESTED {
			// Clear the actionPathCacheMap cache
			actionPathCacheLock.Lock()
			defer actionPathCacheLock.Unlock()
			actionPathCacheMap = map[string]*ActionPathData{}
		}
		return
	})
}

// NewRoute prepares the route to be used in matching.
func NewRoute(moduleSource *Module, method, path, action, fixedArgs, routesPath string, line int) (r *Route) {
	// Handle fixed arguments
	argsReader := strings.NewReader(string(namespaceReplace([]byte(fixedArgs), moduleSource)))
	csvReader := csv.NewReader(argsReader)
	csvReader.TrimLeadingSpace = true
	fargs, err := csvReader.Read()
	if err != nil && err != io.EOF {
		routerLog.Error("NewRoute: Invalid fixed parameters for string ", "error", err, "fixedargs", fixedArgs)
	}

	r = &Route{
		ModuleSource: moduleSource,
		Method:       strings.ToUpper(method),
		Path:         path,
		Action:       string(namespaceReplace([]byte(action), moduleSource)),
		FixedParams:  fargs,
		TreePath:     treePath(strings.ToUpper(method), path),
		routesPath:   routesPath,
		line:         line,
	}

	// URL pattern
	if !strings.HasPrefix(r.Path, "/") {
		routerLog.Error("NewRoute: Absolute URL required.")
		return
	}

	// Ignore the not found status code
	if action != httpStatusCode {
		routerLog.Debugf("NewRoute: New splitActionPath path:%s action:%s", path, action)
		pathData, found := splitActionPath(&ActionPathData{ModuleSource: moduleSource, Route: r}, r.Action, false)
		if found {
			if pathData.TypeOfController != nil {
				// Assign controller type to avoid looking it up based on name
				r.TypeOfController = pathData.TypeOfController
				// Create the fixed parameters
				if l := len(pathData.Route.FixedParams); l > 0 && len(pathData.FixedParamsByName) == 0 {
					methodType := pathData.TypeOfController.Method(pathData.MethodName)
					if methodType != nil {
						pathData.FixedParamsByName = make(map[string]string, l)
						for i, argValue := range pathData.Route.FixedParams {
							Unbind(pathData.FixedParamsByName, methodType.Args[i].Name, argValue)
						}
					} else {
						routerLog.Panicf("NewRoute: Method %s not found for controller %s", pathData.MethodName, pathData.ControllerName)
					}
				}
			}
			r.ControllerNamespace = pathData.ControllerNamespace
			r.ControllerName = pathData.ControllerName
			r.ModuleSource = pathData.ModuleSource
			r.MethodName = pathData.MethodName

			// The same action path could be used for multiple routes (like the Static.Serve)
		} else {
			routerLog.Panicf("NewRoute: Failed to find controller for route path action %s \n%#v\n", path+"?"+r.Action, actionPathCacheMap)
		}
	}
	return
}

func (route *Route) ActionPath() string {
	return route.ModuleSource.Namespace() + route.ControllerName
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
	Module string // The module the route is associated with
	path   string // path to the routes file
}

func (router *Router) Route(req *Request) (routeMatch *RouteMatch) {
	// Override method if set in header
	if method := req.GetHttpHeader("X-HTTP-Method-Override"); method != "" && req.Method == "POST" {
		req.Method = method
	}

	leaf, expansions := router.Tree.Find(treePath(req.Method, req.GetPath()))
	if leaf == nil {
		return nil
	}

	// Create a map of the route parameters.
	var params url.Values
	if len(expansions) > 0 {
		params = make(url.Values)
		for i, v := range expansions {
			params[leaf.Wildcards[i]] = []string{v}
		}
	}
	var route *Route
	var controllerName, methodName string

	// The leaf value is now a list of possible routes to match, only a controller
	routeList := leaf.Value.([]*Route)
	var typeOfController *ControllerType

	//INFO.Printf("Found route for path %s %#v", req.URL.Path, len(routeList))
	for index := range routeList {
		route = routeList[index]
		methodName = route.MethodName

		// Special handling for explicit 404's.
		if route.Action == httpStatusCode {
			route = nil
			break
		}

		// If wildcard match on method name use the method name from the params
		if methodName[0] == ':' {
			if methodKey, found := params[methodName[1:]]; found && len(methodKey) > 0 {
				methodName = strings.ToLower(methodKey[0])
			} else {
				routerLog.Fatal("Route: Failure to find method name in parameters", "params", params, "methodName", methodName)
			}
		}

		// If the action is variablized, replace into it with the captured args.
		controllerName = route.ControllerName
		if controllerName[0] == ':' {
			controllerName = strings.ToLower(params[controllerName[1:]][0])
			if typeOfController = route.ModuleSource.ControllerByName(controllerName, methodName); typeOfController != nil {
				break
			}
		} else {
			typeOfController = route.TypeOfController
			break
		}
		route = nil
	}

	if route == nil {
		routeMatch = notFound
	} else {

		routeMatch = &RouteMatch{
			ControllerName:   route.ControllerNamespace + controllerName,
			MethodName:       methodName,
			Params:           params,
			FixedParams:      route.FixedParams,
			TypeOfController: typeOfController,
			ModuleSource:     route.ModuleSource,
		}
	}

	return
}

// Refresh re-reads the routes file and re-calculates the routing table.
// Returns an error if a specified action could not be found.
func (router *Router) Refresh() (err *Error) {
	RaiseEvent(ROUTE_REFRESH_REQUESTED, nil)
	router.Routes, err = parseRoutesFile(appModule, router.path, "", true)
	RaiseEvent(ROUTE_REFRESH_COMPLETED, nil)
	if err != nil {
		return
	}
	err = router.updateTree()
	return
}

func (router *Router) updateTree() *Error {
	router.Tree = pathtree.New()
	pathMap := map[string][]*Route{}

	allPathsOrdered := []string{}
	// It is possible for some route paths to overlap
	// based on wildcard matches,
	// TODO when pathtree is fixed (made to be smart enough to not require a predefined intake order) keeping the routes in order is not necessary
	for _, route := range router.Routes {
		if _, found := pathMap[route.TreePath]; !found {
			pathMap[route.TreePath] = append(pathMap[route.TreePath], route)
			allPathsOrdered = append(allPathsOrdered, route.TreePath)
		} else {
			pathMap[route.TreePath] = append(pathMap[route.TreePath], route)
		}
	}
	for _, path := range allPathsOrdered {
		routeList := pathMap[path]
		err := router.Tree.Add(path, routeList)

		// Allow GETs to respond to HEAD requests.
		if err == nil && routeList[0].Method == "GET" {
			err = router.Tree.Add(treePath("HEAD", routeList[0].Path), routeList)
		}

		// Error adding a route to the pathtree.
		if err != nil {
			return routeError(err, path, fmt.Sprintf("%#v", routeList), routeList[0].line)
		}
	}
	return nil
}

// Returns the controller namespace and name, action and module if found from the actionPath specified
func splitActionPath(actionPathData *ActionPathData, actionPath string, useCache bool) (pathData *ActionPathData, found bool) {
	actionPath = strings.ToLower(actionPath)
	if pathData, found = actionPathCacheMap[actionPath]; found && useCache {
		return
	}
	var (
		controllerNamespace, controllerName, methodName, action string
		foundModuleSource                                       *Module
		typeOfController                                        *ControllerType
		log                                                     = routerLog.New("actionPath", actionPath)
	)
	actionSplit := strings.Split(actionPath, ".")
	if actionPathData != nil {
		foundModuleSource = actionPathData.ModuleSource
	}
	if len(actionSplit) == 2 {
		controllerName, methodName = strings.ToLower(actionSplit[0]), strings.ToLower(actionSplit[1])
		if i := strings.Index(methodName, "("); i > 0 {
			methodName = methodName[:i]
		}
		log = log.New("controller", controllerName, "method", methodName)
		log.Debug("splitActionPath: Check for namespace")
		if i := strings.Index(controllerName, namespaceSeperator); i > -1 {
			controllerNamespace = controllerName[:i+1]
			if moduleSource, found := ModuleByName(controllerNamespace[:len(controllerNamespace)-1]); found {
				log.Debug("Found module namespace")
				foundModuleSource = moduleSource
				controllerNamespace = moduleSource.Namespace()
			} else {
				log.Warnf("splitActionPath: Unable to find module %s for action: %s", controllerNamespace[:len(controllerNamespace)-1], actionPath)
			}
			controllerName = controllerName[i+1:]
			// Check for the type of controller
			typeOfController = foundModuleSource.ControllerByName(controllerName, methodName)
			found = typeOfController != nil
		} else if controllerName[0] != ':' {
			// First attempt to find the controller in the module source
			if foundModuleSource != nil {
				typeOfController = foundModuleSource.ControllerByName(controllerName, methodName)
				if typeOfController != nil {
					controllerNamespace = typeOfController.Namespace
				}
			}
			log.Info("Found controller for path", "controllerType", typeOfController)

			if typeOfController == nil {
				// Check to see if we can determine the controller from only the controller name
				// an actionPath without a moduleSource will only come from
				// Scan through the controllers
				matchName := controllerName
				for key, controller := range controllers {
					// Strip away the namespace from the controller. to be match
					regularName := key
					if i := strings.Index(key, namespaceSeperator); i > -1 {
						regularName = regularName[i+1:]
					}
					if regularName == matchName {
						// Found controller match
						typeOfController = controller
						controllerNamespace = typeOfController.Namespace
						controllerName = typeOfController.ShortName()
						foundModuleSource = typeOfController.ModuleSource
						found = true
						break
					}
				}
			} else {
				found = true
			}
		} else {
			// If wildcard assign the route the controller namespace found
			controllerNamespace = actionPathData.ModuleSource.Name + namespaceSeperator
			foundModuleSource = actionPathData.ModuleSource
			found = true
		}
		action = actionSplit[1]
	} else {
		foundPaths := ""
		for path := range actionPathCacheMap {
			foundPaths += path + ","
		}
		log.Warnf("splitActionPath: Invalid action path %s found paths %s", actionPath, foundPaths)
		found = false
	}

	// Make sure no concurrent map writes occur
	if found {
		actionPathCacheLock.Lock()
		defer actionPathCacheLock.Unlock()
		if actionPathData != nil {
			actionPathData.ControllerNamespace = controllerNamespace
			actionPathData.ControllerName = controllerName
			actionPathData.MethodName = methodName
			actionPathData.Action = action
			actionPathData.ModuleSource = foundModuleSource
			actionPathData.TypeOfController = typeOfController
		} else {
			actionPathData = &ActionPathData{
				ControllerNamespace: controllerNamespace,
				ControllerName:      controllerName,
				MethodName:          methodName,
				Action:              action,
				ModuleSource:        foundModuleSource,
				TypeOfController:    typeOfController,
			}
		}
		actionPathData.TypeOfController = foundModuleSource.ControllerByName(controllerName, "")
		if actionPathData.TypeOfController == nil && actionPathData.ControllerName[0] != ':' {
			log.Warnf("splitActionPath: No controller found for %s %#v", foundModuleSource.Namespace()+controllerName, controllers)
		}

		pathData = actionPathData
		if pathData.Route != nil && len(pathData.Route.FixedParams) > 0 {
			// If there are fixed params on the route then add them to the path
			// This will give it a unique path and it should still be usable for a reverse lookup provided the name is matchable
			// for example
			// GET   /test/                     Application.Index("Test", "Test2")
			// {{url "Application.Index(test,test)" }}
			// should be parseable
			actionPath = actionPath + "(" + strings.ToLower(strings.Join(pathData.Route.FixedParams, ",")) + ")"
		}
		if actionPathData.Route != nil {
			log.Debugf("splitActionPath: Split Storing recognized action path %s for route  %#v ", actionPath, actionPathData.Route)
		}
		pathData.Key = actionPath
		actionPathCacheMap[actionPath] = pathData
		if !strings.Contains(actionPath, namespaceSeperator) && pathData.TypeOfController != nil {
			actionPathCacheMap[strings.ToLower(pathData.TypeOfController.Namespace)+actionPath] = pathData
			log.Debugf("splitActionPath: Split Storing recognized action path %s for route  %#v ", strings.ToLower(pathData.TypeOfController.Namespace)+actionPath, actionPathData.Route)
		}
	}
	return
}

// parseRoutesFile reads the given routes file and returns the contained routes.
func parseRoutesFile(moduleSource *Module, routesPath, joinedPath string, validate bool) ([]*Route, *Error) {
	contentBytes, err := ioutil.ReadFile(routesPath)
	if err != nil {
		return nil, &Error{
			Title:       "Failed to load routes file",
			Description: err.Error(),
		}
	}
	return parseRoutes(moduleSource, routesPath, joinedPath, string(contentBytes), validate)
}

// parseRoutes reads the content of a routes file into the routing table.
func parseRoutes(moduleSource *Module, routesPath, joinedPath, content string, validate bool) ([]*Route, *Error) {
	var routes []*Route

	// For each line..
	for n, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		const modulePrefix = "module:"

		// Handle included routes from modules.
		// e.g. "module:testrunner" imports all routes from that module.
		if strings.HasPrefix(line, modulePrefix) {
			moduleRoutes, err := getModuleRoutes(line[len(modulePrefix):], joinedPath, validate)
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

		// this will avoid accidental double forward slashes in a route.
		// this also avoids pathtree freaking out and causing a runtime panic
		// because of the double slashes
		if strings.HasSuffix(joinedPath, "/") && strings.HasPrefix(path, "/") {
			joinedPath = joinedPath[0 : len(joinedPath)-1]
		}
		path = strings.Join([]string{AppRoot, joinedPath, path}, "")

		// This will import the module routes under the path described in the
		// routes file (joinedPath param). e.g. "* /jobs module:jobs" -> all
		// routes' paths will have the path /jobs prepended to them.
		// See #282 for more info
		if method == "*" && strings.HasPrefix(action, modulePrefix) {
			moduleRoutes, err := getModuleRoutes(action[len(modulePrefix):], path, validate)
			if err != nil {
				return nil, routeError(err, routesPath, content, n)
			}
			routes = append(routes, moduleRoutes...)
			continue
		}

		route := NewRoute(moduleSource, method, path, action, fixedArgs, routesPath, n)
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
	if route.Action == httpStatusCode {
		return nil
	}

	// Skip variable routes.
	if route.ControllerName[0] == ':' || route.MethodName[0] == ':' {
		return nil
	}

	// Precheck to see if controller exists
	if _, found := controllers[route.ControllerNamespace+route.ControllerName]; !found {
		// Scan through controllers to find module
		for _, c := range controllers {
			controllerName := strings.ToLower(c.Type.Name())
			if controllerName == route.ControllerName {
				route.ControllerNamespace = c.ModuleSource.Name + namespaceSeperator
				routerLog.Warn("validateRoute: Matched empty namespace route for %s to this namespace %s for the route %s", controllerName, c.ModuleSource.Name, route.Path)
			}
		}
	}

	// TODO need to check later
	// does it do only validation or validation and instantiate the controller.
	var c Controller
	return c.SetTypeAction(route.ControllerNamespace+route.ControllerName, route.MethodName, route.TypeOfController)
}

// routeError adds context to a simple error message.
func routeError(err error, routesPath, content string, n int) *Error {
	if revelError, ok := err.(*Error); ok {
		return revelError
	}
	// Load the route file content if necessary
	if content == "" {
		if contentBytes, er := ioutil.ReadFile(routesPath); er != nil {
			routerLog.Error("routeError: Failed to read route file ", "file", routesPath, "error", er)
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
		Stack:       fmt.Sprintf("%s", logger.NewCallStack()),
	}
}

// getModuleRoutes loads the routes file for the given module and returns the
// list of routes.
func getModuleRoutes(moduleName, joinedPath string, validate bool) (routes []*Route, err *Error) {
	// Look up the module.  It may be not found due to the common case of e.g. the
	// testrunner module being active only in dev mode.
	module, found := ModuleByName(moduleName)
	if !found {
		routerLog.Debug("getModuleRoutes: Skipping routes for inactive module", "module", moduleName)
		return nil, nil
	}
	routePath := filepath.Join(module.Path, "conf", "routes")
	if _, e := os.Stat(routePath); e == nil {
		routes, err = parseRoutesFile(module, routePath, joinedPath, validate)
	}
	if err == nil {
		for _, route := range routes {
			route.ModuleSource = module
		}
	}

	return routes, err
}

// Groups:
// 1: method
// 4: path
// 5: action
// 6: fixedargs
var routePattern = regexp.MustCompile(
	"(?i)^(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD|WS|PROPFIND|MKCOL|COPY|MOVE|PROPPATCH|LOCK|UNLOCK|TRACE|PURGE|\\*)" +
		"[(]?([^)]*)(\\))?[ \t]+" +
		"(.*/[^ \t]*)[ \t]+([^ \t(]+)" +
		`\(?([^)]*)\)?[ \t]*$`)

func parseRouteLine(line string) (method, path, action, fixedArgs string, found bool) {
	matches := routePattern.FindStringSubmatch(line)
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
	Host, Method, URL, Action string
	Star                      bool
	Args                      map[string]string
}

func (a *ActionDefinition) String() string {
	return a.URL
}

func (router *Router) Reverse(action string, argValues map[string]string) (ad *ActionDefinition) {
	log := routerLog.New("action", action)
	pathData, found := splitActionPath(nil, action, true)
	if !found {
		routerLog.Error("splitActionPath: Failed to find reverse route", "action", action, "arguments", argValues)
		return nil
	}

	log.Debug("Checking for route", "pathdataRoute", pathData.Route)
	if pathData.Route == nil {
		var possibleRoute *Route
		// If the route is nil then we need to go through the routes to find the first matching route
		// from this controllers namespace, this is likely a wildcard route match
		for _, route := range router.Routes {
			// Skip routes that are not wild card or empty
			if route.ControllerName == "" || route.MethodName == "" {
				continue
			}
			if route.ModuleSource == pathData.ModuleSource && route.ControllerName[0] == ':' {
				// Wildcard match in same module space
				pathData.Route = route
				break
			} else if route.ActionPath() == pathData.ModuleSource.Namespace()+pathData.ControllerName &&
				(route.Method[0] == ':' || route.Method == pathData.MethodName) {
				// Action path match
				pathData.Route = route
				break
			} else if route.ControllerName == pathData.ControllerName &&
				(route.Method[0] == ':' || route.Method == pathData.MethodName) {
				// Controller name match
				possibleRoute = route
			}
		}
		if pathData.Route == nil && possibleRoute != nil {
			pathData.Route = possibleRoute
			routerLog.Warnf("Reverse: For a url reverse a match was based on  %s matched path to route %#v ", action, possibleRoute)
		}
		if pathData.Route != nil {
			routerLog.Debugf("Reverse: Reverse Storing recognized action path %s for route %#v\n", action, pathData.Route)
		}
	}

	// Likely unknown route because of a wildcard, perform manual lookup
	if pathData.Route != nil {
		route := pathData.Route

		// If the controller or method are wildcards we need to populate the argValues
		controllerWildcard := route.ControllerName[0] == ':'
		methodWildcard := route.MethodName[0] == ':'

		// populate route arguments with the names
		if controllerWildcard {
			argValues[route.ControllerName[1:]] = pathData.ControllerName
		}
		if methodWildcard {
			argValues[route.MethodName[1:]] = pathData.MethodName
		}
		// In theory all routes should be defined and pre-populated, the route controllers may not be though
		// with wildcard routes
		if pathData.TypeOfController == nil {
			if controllerWildcard || methodWildcard {
				if controller := ControllerTypeByName(pathData.ControllerNamespace+pathData.ControllerName, route.ModuleSource); controller != nil {
					// Wildcard match boundary
					pathData.TypeOfController = controller
					// See if the path exists in the module based
				} else {
					routerLog.Errorf("Reverse: Controller %s not found in reverse lookup", pathData.ControllerNamespace+pathData.ControllerName)
					return
				}
			}
		}

		if pathData.TypeOfController == nil {
			routerLog.Errorf("Reverse: Controller %s not found in reverse lookup", pathData.ControllerNamespace+pathData.ControllerName)
			return
		}
		var (
			queryValues  = make(url.Values)
			pathElements = strings.Split(route.Path, "/")
		)
		for i, el := range pathElements {
			if el == "" || (el[0] != ':' && el[0] != '*') {
				continue
			}
			val, ok := pathData.FixedParamsByName[el[1:]]
			if !ok {
				val, ok = argValues[el[1:]]
			}
			if !ok {
				val = "<nil>"
				routerLog.Error("Reverse: reverse route missing route argument ", "argument", el[1:])
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
		urlPath := strings.Join(pathElements, "/")
		if len(queryValues) > 0 {
			urlPath += "?" + queryValues.Encode()
		}

		method := route.Method
		star := false
		if route.Method == "*" {
			method = "GET"
			star = true
		}

		//INFO.Printf("Reversing action %s to %s Using Route %#v",action,url,pathData.Route)

		return &ActionDefinition{
			URL:    urlPath,
			Method: method,
			Star:   star,
			Action: action,
			Args:   argValues,
			Host:   "TODO",
		}
	}

	routerLog.Error("Reverse: Failed to find controller for reverse route", "action", action, "arguments", argValues)
	return nil
}

func RouterFilter(c *Controller, fc []Filter) {
	// Figure out the Controller/Action
	route := MainRouter.Route(c.Request)
	if route == nil {
		c.Result = c.NotFound("No matching route found: " + c.Request.GetRequestURI())
		return
	}

	// The route may want to explicitly return a 404.
	if route.Action == httpStatusCode {
		c.Result = c.NotFound("(intentionally)")
		return
	}

	// Set the action.
	if err := c.SetTypeAction(route.ControllerName, route.MethodName, route.TypeOfController); err != nil {
		c.Result = c.NotFound(err.Error())
		return
	}

	// Add the route and fixed params to the Request Params.
	c.Params.Route = route.Params
	// Assign logger if from module
	if c.Type.ModuleSource != nil && c.Type.ModuleSource != appModule {
		c.Log = c.Type.ModuleSource.Log.New("ip", c.ClientIP,
			"path", c.Request.URL.Path, "method", c.Request.Method)
	}

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
			routerLog.Warn("RouterFilter: Too many parameters to action", "action", route.Action, "value", value)
			break
		}
	}

	fc[0](c, fc[1:])
}

// HTTPMethodOverride overrides allowed http methods via form or browser param
func HTTPMethodOverride(c *Controller, fc []Filter) {
	// An array of HTTP verbs allowed.
	verbs := []string{"POST", "PUT", "PATCH", "DELETE"}

	method := strings.ToUpper(c.Request.Method)

	if method == "POST" {
		param := ""
		if f, err := c.Request.GetForm(); err == nil {
			param = strings.ToUpper(f.Get("_method"))
		}

		if len(param) > 0 {
			override := false
			// Check if param is allowed
			for _, verb := range verbs {
				if verb == param {
					override = true
					break
				}
			}

			if override {
				c.Request.Method = param
			} else {
				c.Response.Status = 405
				c.Result = c.RenderError(&Error{
					Title:       "Method not allowed",
					Description: "Method " + param + " is not allowed (valid: " + strings.Join(verbs, ", ") + ")",
				})
				return
			}

		}
	}

	fc[0](c, fc[1:]) // Execute the next filter stage.
}

func init() {
	OnAppStart(func() {
		MainRouter = NewRouter(filepath.Join(BasePath, "conf", "routes"))
		err := MainRouter.Refresh()
		if MainWatcher != nil && Config.BoolDefault("watch.routes", true) {
			MainWatcher.Listen(MainRouter, MainRouter.path)
		} else if err != nil {
			// Not in dev mode and Route loading failed, we should crash.
			routerLog.Panic("init: router initialize error", "error", err)
		}
	})
}
