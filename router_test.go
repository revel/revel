package revel

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
		Method:        "GET",
		Path:          "/",
		Action:        "Application.Index",
		pathPattern:   regexp.MustCompile("/$"),
		args:          []*arg{},
		FixedParams:   []string{},
		actionPattern: regexp.MustCompile("Application\\.Index"),
	},

	"post /app/{id} Application.SaveApp": &Route{
		Method:      "POST",
		Path:        "/app/{id}",
		Action:      "Application.SaveApp",
		pathPattern: regexp.MustCompile("/app/(?P<id>[^/]+)$"),
		args: []*arg{
			{
				name:       "id",
				constraint: regexp.MustCompile("[^/]+"),
			},
		},
		FixedParams:   []string{},
		actionPattern: regexp.MustCompile("Application\\.SaveApp"),
	},

	"post /app/{<[0-9]+>id} Application.SaveApp": &Route{
		Method:      "POST",
		Path:        "/app/{<[0-9]+>id}",
		Action:      "Application.SaveApp",
		pathPattern: regexp.MustCompile("/app/(?P<id>[0-9]+)$"),
		args: []*arg{
			{
				name:       "id",
				constraint: regexp.MustCompile("[0-9]+"),
			},
		},
		FixedParams:   []string{},
		actionPattern: regexp.MustCompile("Application\\.SaveApp"),
	},

	"get /app/? Application.List": &Route{
		Method:        "GET",
		Path:          "/app/?",
		Action:        "Application.List",
		pathPattern:   regexp.MustCompile("/app/?$"),
		args:          []*arg{},
		FixedParams:   []string{},
		actionPattern: regexp.MustCompile("Application\\.List"),
	},

	`get /apps/{<\d+>appId}/? Application.Show`: &Route{
		Method:      "GET",
		Path:        `/apps/{<\d+>appId}/?`,
		Action:      "Application.Show",
		pathPattern: regexp.MustCompile(`/apps/(?P<appId>\d+)/?$`),
		args: []*arg{
			{
				name:       "appId",
				constraint: regexp.MustCompile(`\d+`),
			},
		},
		FixedParams:   []string{},
		actionPattern: regexp.MustCompile("Application\\.Show"),
	},

	`GET /public/{<.+>filepath}   Static.Serve("public")`: &Route{
		Method:      "GET",
		Path:        "/public/{<.+>filepath}",
		Action:      "Static.Serve",
		pathPattern: regexp.MustCompile(`/public/(?P<filepath>.+)$`),
		args: []*arg{
			{
				name:       "filepath",
				constraint: regexp.MustCompile(`.+`),
			},
		},
		FixedParams: []string{
			"public",
		},
		actionPattern: regexp.MustCompile("Static\\.Serve"),
	},

	`GET /javascript/{<.+>filepath} Static.Serve("public/js")`: &Route{
		Method:      "GET",
		Path:        "/javascript/{<.+>filepath}",
		Action:      "Static.Serve",
		pathPattern: regexp.MustCompile(`/javascript/(?P<filepath>.+)$`),
		args: []*arg{
			{
				name:       "filepath",
				constraint: regexp.MustCompile(`.+`),
			},
		},
		FixedParams: []string{
			"public",
		},
		actionPattern: regexp.MustCompile("Static\\.Serve"),
	},

	"* /apps/{id}/{action} Application.{action}": &Route{
		Method:      "*",
		Path:        "/apps/{id}/{action}",
		Action:      "Application.{action}",
		pathPattern: regexp.MustCompile("/apps/(?P<id>[^/]+)/(?P<action>[^/]+)$"),
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
		FixedParams:   []string{},
		actionPattern: regexp.MustCompile("Application\\.(?P<action>[^/]+)"),
	},

	"* /{controller}/{action} {controller}.{action}": &Route{
		Method:      "*",
		Path:        "/{controller}/{action}",
		Action:      "{controller}.{action}",
		pathPattern: regexp.MustCompile("/(?P<controller>[^/]+)/(?P<action>[^/]+)$"),
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
		FixedParams:   []string{},
		actionPattern: regexp.MustCompile("(?P<controller>[^/]+)\\.(?P<action>[^/]+)"),
	},
}

// Run the test cases above.
func TestComputeRoute(t *testing.T) {
	for routeLine, expected := range routeTestCases {
		method, path, action, fixedArgs, found := parseRouteLine(routeLine)
		if !found {
			t.Error("Failed to parse route line:", routeLine)
			continue
		}
		actual := NewRoute(method, path, action, fixedArgs)
		eq(t, "Method", actual.Method, expected.Method)
		eq(t, "Path", actual.Path, expected.Path)
		eq(t, "Action", actual.Action, expected.Action)
		eq(t, "pathPattern", fmt.Sprint(actual.pathPattern), fmt.Sprint(expected.pathPattern))
		eq(t, "len(args)", len(actual.args), len(expected.args))
		for i, arg := range actual.args {
			if len(expected.args) <= i {
				break
			}
			eq(t, "arg.name", arg.name, expected.args[i].name)
			eq(t, "arg.constraint", arg.constraint.String(), expected.args[i].constraint.String())
		}
		eq(t, "actionPattern", fmt.Sprint(actual.actionPattern), fmt.Sprint(expected.actionPattern))
		if t.Failed() {
			t.Fatal("Failed on route:", routeLine)
		}
	}
}

// Router Tests

const TEST_ROUTES = `
# This is a comment
GET   /                          Application.Index
GET   /app/{id}/?                Application.Show
POST  /app/{id}                  Application.Save
PATCH /app/{id}/?                Application.Update
GET   /javascript/{<.+>filepath} Static.Serve("public/js")
GET   /public/{<.+>filepath}     Static.Serve("public")
*     /{controller}/{action}     {controller}.{action}

GET   /favicon.ico               404
`

var routeMatchTestCases = map[*http.Request]*RouteMatch{
	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Index",
		FixedParams:    []string{},
		Params:         map[string]string{},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/app/123"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Show",
		FixedParams:    []string{},
		Params:         map[string]string{"id": "123"},
	},

	&http.Request{
		Method: "PATCH",
		URL:    &url.URL{Path: "/app/123"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Update",
		FixedParams:    []string{},
		Params:         map[string]string{"id": "123"},
	},

	&http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/app/123"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Save",
		FixedParams:    []string{},
		Params:         map[string]string{"id": "123"},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/app/123/"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Show",
		FixedParams:    []string{},
		Params:         map[string]string{"id": "123"},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/public/style.css"},
	}: &RouteMatch{
		ControllerName: "Static",
		MethodName:     "Serve",
		FixedParams:    []string{"public"},
		Params:         map[string]string{"filepath": "style.css"},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/javascript/sessvars.js"},
	}: &RouteMatch{
		ControllerName: "Static",
		MethodName:     "Serve",
		FixedParams:    []string{"public"},
		Params:         map[string]string{"filepath": "sessvars.js"},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/Implicit/Route"},
	}: &RouteMatch{
		ControllerName: "Implicit",
		MethodName:     "Route",
		FixedParams:    []string{},
		Params:         map[string]string{"controller": "Implicit", "action": "Route"},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/favicon.ico"},
	}: &RouteMatch{
		ControllerName: "",
		MethodName:     "",
		Action:         "404",
		FixedParams:    []string{},
		Params:         map[string]string{},
	},
}

func TestRouteMatches(t *testing.T) {
	BasePath = "/BasePath"
	router := NewRouter("")
	router.Routes, _ = parseRoutes("", TEST_ROUTES, false)
	for req, expected := range routeMatchTestCases {
		t.Log("Routing:", req.Method, req.URL)
		actual := router.Route(req)
		if !eq(t, "Found route", actual != nil, expected != nil) {
			continue
		}
		eq(t, "ControllerName", actual.ControllerName, expected.ControllerName)
		eq(t, "MethodName", actual.MethodName, expected.MethodName)
		eq(t, "len(Params)", len(actual.Params), len(expected.Params))
		for key, actualValue := range actual.Params {
			eq(t, "Params", actualValue, expected.Params[key])
		}
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
		Url:    "/app/123/",
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
	router := NewRouter("")
	router.Routes, _ = parseRoutes("", TEST_ROUTES, false)
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

func BenchmarkRouter(b *testing.B) {
	router := NewRouter("")
	router.Routes, _ = parseRoutes("", TEST_ROUTES, false)
	b.ResetTimer()
	for i := 0; i < b.N/len(routeMatchTestCases); i++ {
		for req, _ := range routeMatchTestCases {
			router.Route(req)
		}
	}
}

func BenchmarkRouterFilter(b *testing.B) {
	startFakeBookingApp()
	controllers := []*Controller{
		{Request: NewRequest(showRequest)},
		{Request: NewRequest(staticRequest)},
	}
	for _, c := range controllers {
		c.Params = ParseParams(c.Request)
	}

	b.ResetTimer()
	for i := 0; i < b.N/len(controllers); i++ {
		for _, c := range controllers {
			RouterFilter.Call(c, NilChain)
		}
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
