package revel

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Data-driven tests that check that a given routes-file line translates into
// the expected Route object.
var routeTestCases = map[string]*Route{
	"get / Application.Index": &Route{
		Method:      "GET",
		Path:        "/",
		Action:      "Application.Index",
		FixedParams: []string{},
	},

	"post /app/:id Application.SaveApp": &Route{
		Method:      "POST",
		Path:        "/app/:id",
		Action:      "Application.SaveApp",
		FixedParams: []string{},
	},

	"get /app/ Application.List": &Route{
		Method:      "GET",
		Path:        "/app/",
		Action:      "Application.List",
		FixedParams: []string{},
	},

	`get /app/:appId/ Application.Show`: &Route{
		Method:      "GET",
		Path:        `/app/:appId/`,
		Action:      "Application.Show",
		FixedParams: []string{},
	},

	`get /app-wild/*appId/ Application.WildShow`: &Route{
		Method:      "GET",
		Path:        `/app-wild/*appId/`,
		Action:      "Application.WildShow",
		FixedParams: []string{},
	},

	`GET /public/:filepath   Static.Serve("public")`: &Route{
		Method: "GET",
		Path:   "/public/:filepath",
		Action: "Static.Serve",
		FixedParams: []string{
			"public",
		},
	},

	`GET /javascript/:filepath Static.Serve("public/js")`: &Route{
		Method: "GET",
		Path:   "/javascript/:filepath",
		Action: "Static.Serve",
		FixedParams: []string{
			"public",
		},
	},

	"* /apps/:id/:action Application.:action": &Route{
		Method:      "*",
		Path:        "/apps/:id/:action",
		Action:      "Application.:action",
		FixedParams: []string{},
	},

	"* /:controller/:action :controller.:action": &Route{
		Method:      "*",
		Path:        "/:controller/:action",
		Action:      ":controller.:action",
		FixedParams: []string{},
	},

	`GET / Application.Index("Test", "Test2")`: &Route{
		Method: "GET",
		Path:   "/",
		Action: "Application.Index",
		FixedParams: []string{
			"Test",
			"Test2",
		},
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
		actual := NewRoute(method, path, action, fixedArgs, "", 0)
		eq(t, "Method", actual.Method, expected.Method)
		eq(t, "Path", actual.Path, expected.Path)
		eq(t, "Action", actual.Action, expected.Action)
		if t.Failed() {
			t.Fatal("Failed on route:", routeLine)
		}
	}
}

// Router Tests

const TEST_ROUTES = `
# This is a comment
GET   /                          Application.Index
GET   /test/                     Application.Index("Test", "Test2")
GET   /app/:id/                  Application.Show
GET   /app-wild/*id/             Application.WildShow
POST  /app/:id                   Application.Save
PATCH /app/:id/                  Application.Update
GET   /javascript/:filepath      Static.Serve("public/js")
GET   /public/*filepath          Static.Serve("public")
*     /:controller/:action       :controller.:action

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
		Params:         map[string][]string{},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/test/"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Index",
		FixedParams:    []string{"Test", "Test2"},
		Params:         map[string][]string{},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/app/123"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Show",
		FixedParams:    []string{},
		Params:         map[string][]string{"id": {"123"}},
	},

	&http.Request{
		Method: "PATCH",
		URL:    &url.URL{Path: "/app/123"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Update",
		FixedParams:    []string{},
		Params:         map[string][]string{"id": {"123"}},
	},

	&http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/app/123"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Save",
		FixedParams:    []string{},
		Params:         map[string][]string{"id": {"123"}},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/app/123/"},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Show",
		FixedParams:    []string{},
		Params:         map[string][]string{"id": {"123"}},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/public/css/style.css"},
	}: &RouteMatch{
		ControllerName: "Static",
		MethodName:     "Serve",
		FixedParams:    []string{"public"},
		Params:         map[string][]string{"filepath": {"css/style.css"}},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/javascript/sessvars.js"},
	}: &RouteMatch{
		ControllerName: "Static",
		MethodName:     "Serve",
		FixedParams:    []string{"public/js"},
		Params:         map[string][]string{"filepath": {"sessvars.js"}},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/Implicit/Route"},
	}: &RouteMatch{
		ControllerName: "Implicit",
		MethodName:     "Route",
		FixedParams:    []string{},
		Params: map[string][]string{
			"METHOD":     {"GET"},
			"controller": {"Implicit"},
			"action":     {"Route"},
		},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/favicon.ico"},
	}: &RouteMatch{
		ControllerName: "",
		MethodName:     "",
		Action:         "404",
		FixedParams:    []string{},
		Params:         map[string][]string{},
	},

	&http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/app/123"},
		Header: http.Header{"X-Http-Method-Override": []string{"PATCH"}},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Update",
		FixedParams:    []string{},
		Params:         map[string][]string{"id": {"123"}},
	},

	&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/app/123"},
		Header: http.Header{"X-Http-Method-Override": []string{"PATCH"}},
	}: &RouteMatch{
		ControllerName: "Application",
		MethodName:     "Show",
		FixedParams:    []string{},
		Params:         map[string][]string{"id": {"123"}},
	},
}

func TestRouteMatches(t *testing.T) {
	BasePath = "/BasePath"
	router := NewRouter("")
	router.Routes, _ = parseRoutes("", "", TEST_ROUTES, false)
	router.updateTree()
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
			eq(t, "Params", actualValue[0], expected.Params[key][0])
		}
		eq(t, "len(FixedParams)", len(actual.FixedParams), len(expected.FixedParams))
		for i, actualValue := range actual.FixedParams {
			eq(t, "FixedParams", actualValue, expected.FixedParams[i])
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

	&ReverseRouteArgs{
		action: "Application.WildShow",
		args:   map[string]string{"id": "123"},
	}: &ActionDefinition{
		Url:    "/app-wild/123/",
		Method: "GET",
		Star:   false,
		Action: "Application.WildShow",
	},
}

func TestReverseRouting(t *testing.T) {
	router := NewRouter("")
	router.Routes, _ = parseRoutes("", "", TEST_ROUTES, false)
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
	router.Routes, _ = parseRoutes("", "", TEST_ROUTES, false)
	router.updateTree()
	b.ResetTimer()
	for i := 0; i < b.N/len(routeMatchTestCases); i++ {
		for req, _ := range routeMatchTestCases {
			r := router.Route(req)
			if r == nil {
				b.Errorf("Request not found: %s", req.URL.Path)
			}
		}
	}
}

// The benchmark from github.com/ant0ine/go-urlrouter
func BenchmarkLargeRouter(b *testing.B) {
	router := NewRouter("")

	routePaths := []string{
		"/",
		"/signin",
		"/signout",
		"/profile",
		"/settings",
		"/upload/*file",
	}
	for i := 0; i < 10; i++ {
		for j := 0; j < 5; j++ {
			routePaths = append(routePaths, fmt.Sprintf("/resource%d/:id/property%d", i, j))
		}
		routePaths = append(routePaths, fmt.Sprintf("/resource%d/:id", i))
		routePaths = append(routePaths, fmt.Sprintf("/resource%d", i))
	}
	routePaths = append(routePaths, "/:any")

	for _, p := range routePaths {
		router.Routes = append(router.Routes,
			NewRoute("GET", p, "Controller.Action", "", "", 0))
	}
	router.updateTree()

	requestUrls := []string{
		"http://example.org/",
		"http://example.org/resource9/123",
		"http://example.org/resource9/123/property1",
		"http://example.org/doesnotexist",
	}
	var reqs []*http.Request
	for _, url := range requestUrls {
		req, _ := http.NewRequest("GET", url, nil)
		reqs = append(reqs, req)
	}

	b.ResetTimer()

	for i := 0; i < b.N/len(reqs); i++ {
		for _, req := range reqs {
			route := router.Route(req)
			if route == nil {
				b.Errorf("Failed to route: %s", req.URL.Path)
			}
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
		c.Params = &Params{}
		ParseParams(c.Params, c.Request)
	}

	b.ResetTimer()
	for i := 0; i < b.N/len(controllers); i++ {
		for _, c := range controllers {
			RouterFilter(c, NilChain)
		}
	}
}

func TestOverrideMethodFilter(t *testing.T) {
	req, _ := http.NewRequest("POST", "/hotels/3", strings.NewReader("_method=put"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	c := Controller{
		Request: NewRequest(req),
	}

	if HttpMethodOverride(&c, NilChain); c.Request.Request.Method != "PUT" {
		t.Errorf("Expected to override current method '%s' in route, found '%s' instead", "", c.Request.Request.Method)
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
