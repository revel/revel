package rev

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"testing"
)

// Data-driven tests that check that a given routes-file line translates into
// the expected Route object.
var routeTestCases = map[string]*Route{
	"get / Application.Index": &Route{
		method:        "GET",
		path:          "/",
		action:        "Application.Index",
		pathPattern:   regexp.MustCompile("/$"),
		staticDir:     "",
		args:          []*arg{},
		actionPattern: regexp.MustCompile("Application\\.Index"),
	},

	"post /app/{id} Application.SaveApp": &Route{
		method:      "POST",
		path:        "/app/{id}",
		action:      "Application.SaveApp",
		pathPattern: regexp.MustCompile("/app/(?P<id>[^/]+)$"),
		staticDir:   "",
		args: []*arg{
			{
				name:       "id",
				constraint: regexp.MustCompile("[^/]+"),
			},
		},
		actionPattern: regexp.MustCompile("Application\\.SaveApp"),
	},

	"post /app/{<[0-9]+>id} Application.SaveApp": &Route{
		method:      "POST",
		path:        "/app/{<[0-9]+>id}",
		action:      "Application.SaveApp",
		pathPattern: regexp.MustCompile("/app/(?P<id>[0-9]+)$"),
		staticDir:   "",
		args: []*arg{
			{
				name:       "id",
				constraint: regexp.MustCompile("[0-9]+"),
			},
		},
		actionPattern: regexp.MustCompile("Application\\.SaveApp"),
	},

	"get /app/? Application.List": &Route{
		method:      "GET",
		path:        "/app/?",
		action:      "Application.List",
		pathPattern: regexp.MustCompile("/app/?$"),
		staticDir:   "",
		args: []*arg{},
		actionPattern: regexp.MustCompile("Application\\.List"),
	},

	"GET /public/ staticDir:www": &Route{
		method:        "GET",
		path:          "/public/",
		action:        "staticDir:www",
		pathPattern:   regexp.MustCompile("^/public/(.*)$"),
		staticDir:     "www",
		args:          []*arg{},
		actionPattern: nil,
	},

	"* /apps/{id}/{action} Application.{action}": &Route{
		method:      "*",
		path:        "/apps/{id}/{action}",
		action:      "Application.{action}",
		pathPattern: regexp.MustCompile("/apps/(?P<id>[^/]+)/(?P<action>[^/]+)$"),
		staticDir:   "",
		args: []*arg{
			{
				name:       "id",
				constraint: regexp.MustCompile("[^/]+"),
			},
			{
				name:       "action",
				constraint: regexp.MustCompile("[^/]+"),
			},
		},
		actionPattern: regexp.MustCompile("Application\\.(?P<action>[^/]+)"),
		actionArgs:    []string{"action"},
	},

	"* /{controller}/{action} {controller}.{action}": &Route{
		method:      "*",
		path:        "/{controller}/{action}",
		action:      "{controller}.{action}",
		pathPattern: regexp.MustCompile("/(?P<controller>[^/]+)/(?P<action>[^/]+)$"),
		staticDir:   "",
		args: []*arg{
			{
				name:       "controller",
				constraint: regexp.MustCompile("[^/]+"),
			},
			{
				name:       "action",
				constraint: regexp.MustCompile("[^/]+"),
			},
		},
		actionPattern: regexp.MustCompile("(?P<controller>[^/]+)\\.(?P<action>[^/]+)"),
		actionArgs:    []string{"controller", "action"},
	},
}

// Run the test cases above.
func TestComputeRoute(t *testing.T) {
	for routeLine, expected := range routeTestCases {
		method, path, action, found := parseRouteLine(routeLine)
		if !found {
			t.Error("Failed to parse route line:", routeLine)
			continue
		}
		actual := NewRoute(method, path, action)
		eq(t, "Method", actual.method, expected.method)
		eq(t, "Path", actual.path, expected.path)
		eq(t, "Action", actual.action, expected.action)
		eq(t, "pathPattern", fmt.Sprint(actual.pathPattern), fmt.Sprint(expected.pathPattern))
		eq(t, "staticDir", actual.staticDir, expected.staticDir)
		eq(t, "len(args)", len(actual.args), len(expected.args))
		for i, arg := range actual.args {
			if len(expected.args) <= i {
				break
			}
			eq(t, "arg.name", arg.name, expected.args[i].name)
			eq(t, "arg.constraint", arg.constraint.String(), expected.args[i].constraint.String())
		}
		eq(t, "actionPattern", fmt.Sprint(actual.actionPattern), fmt.Sprint(expected.actionPattern))
		eq(t, "len(actionArgs)", len(actual.actionArgs), len(expected.actionArgs))
		eq(t, "actionArgs", fmt.Sprint(actual.actionArgs), fmt.Sprint(expected.actionArgs))
		if t.Failed() {
			t.Fatal("Failed on route:", routeLine)
		}
	}
}

// Router Tests

const TEST_ROUTES = `
# This is a comment
GET  /                       Application.Index
GET  /app/{id}               Application.Show
POST /app/{id}               Application.Save

GET  /public/                staticDir:www
*    /{controller}/{action} {controller}.{action}
`

var routeMatchTestCases = map[*http.Request]*RouteMatch{
	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Index",
		Params:         map[string]string{},
		StaticFilename: "",
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/app/123"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "ShowApp",
		Params:         map[string]string{"id": "123"},
		StaticFilename: "",
	},

	&http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/app/123"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "SaveApp",
		Params:         map[string]string{"id": "123"},
		StaticFilename: "",
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/public/style.css"},
	}: &RouteMatch{
		ControllerName: "",
		MethodName:     "",
		Params:         map[string]string{},
		StaticFilename: "www/style.css",
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/Implicit/Route"},
	}: &RouteMatch{
		ControllerName: "Implicit",
		MethodName:     "Route",
		Params:         map[string]string{"controller": "Implicit", "action": "Route"},
		StaticFilename: "",
	},
}

func TestRouteMatches(t *testing.T) {
	router := NewRouter(TEST_ROUTES)
	for req, expected := range routeMatchTestCases {
		actual := router.Route(req)
		if !eq(t, "Found route", actual != nil, expected != nil) {
			continue
		}
		eq(t, "ControllerName", actual.ControllerName, expected.ControllerName)
		eq(t, "MethodName", actual.ControllerName, expected.ControllerName)
		eq(t, "len(Params)", len(actual.Params), len(expected.Params))
		for key, actualValue := range actual.Params {
			eq(t, "Params", actualValue, expected.Params[key])
		}
		eq(t, "StaticFilename", actual.StaticFilename, expected.StaticFilename)
	}
}

// Reverse Routing

type ReverseRouteArgs struct {
	action string
	args   map[string]string
}

var reverseRoutingTestCases = map[*ReverseRouteArgs]*ActionDefinition{
	&ReverseRouteArgs{
		action: "Application.Index",
		args:   map[string]string{},
	}: &ActionDefinition{
		Url:    "/",
		Method: "GET",
		Star:   false,
		Action: "Application.Index",
	},

	&ReverseRouteArgs{
		action: "Application.Show",
		args:   map[string]string{"id": "123"},
	}: &ActionDefinition{
		Url:    "/app/123",
		Method: "GET",
		Star:   false,
		Action: "Application.Show",
	},

	&ReverseRouteArgs{
		action: "Implicit.Route",
		args:   map[string]string{},
	}: &ActionDefinition{
		Url:    "/Implicit/Route",
		Method: "GET",
		Star:   true,
		Action: "Implicit.Route",
	},

	&ReverseRouteArgs{
		action: "Application.Save",
		args:   map[string]string{"id": "123", "c": "http://continue"},
	}: &ActionDefinition{
		Url:    "/app/123?c=http%3A%2F%2Fcontinue",
		Method: "POST",
		Star:   false,
		Action: "Application.Save",
	},
}

func TestReverseRouting(t *testing.T) {
	router := NewRouter(TEST_ROUTES)
	for routeArgs, expected := range reverseRoutingTestCases {
		actual := router.Reverse(routeArgs.action, routeArgs.args)
		if !eq(t, "Found route", actual != nil, expected != nil) {
			continue
		}
		eq(t, "Url", actual.Url, expected.Url)
		eq(t, "Method", actual.Method, expected.Method)
		eq(t, "Star", actual.Star, expected.Star)
		eq(t, "Action", actual.Action, expected.Action)
	}
}

// Helpers

func eq(t *testing.T, name string, a, b interface{}) bool {
	if a != b {
		t.Error(name, ": (actual)", a, " != ", b, "(expected)")
		return false
	}
	return true
}
