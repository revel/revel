package play

import (
	"fmt"
	"regexp"
	"testing"
)

// Data-driven tests that check that a given routes-file line translates into
// the expected Route object.
var routeTestCases = map[string]*Route{
	"get / Application.Index": &Route{
		method:"GET",
		path:"/",
		action:"Application.Index",
		pathPattern: regexp.MustCompile("/$"),
		staticDir: "",
		args: []*arg{},
		actionPattern: regexp.MustCompile("Application\\.Index"),
	},

	"post /app/{id} Application.SaveApp": &Route{
		method:"POST",
		path:"/app/{id}",
		action:"Application.SaveApp",
		pathPattern: regexp.MustCompile("/app/(?P<id>[^/]+)$"),
		staticDir: "",
		args: []*arg{
			{
				name: "id",
				constraint: regexp.MustCompile("[^/]+"),
			},
		},
		actionPattern: regexp.MustCompile("Application\\.SaveApp"),
	},

	"GET /public/ staticDir:www": &Route{
		method:"GET",
		path:"/public/",
		action:"staticDir:www",
		pathPattern: regexp.MustCompile("^/public/(.*)$"),
		staticDir: "www",
		args: []*arg{},
		actionPattern: nil,
	},

	"* /{controller}/{action} {controller}.{action}": &Route{
		method:"*",
		path:"/{controller}/{action}",
		action:"{controller}.{action}",
		pathPattern: regexp.MustCompile("/(?P<controller>[^/]+)/(?P<action>[^/]+)$"),
		staticDir: "",
		args: []*arg{
			{
				name: "controller",
				constraint: regexp.MustCompile("[^/]+"),
			},
			{
				name: "action",
				constraint: regexp.MustCompile("[^/]+"),
			},
		},
		actionPattern: regexp.MustCompile("({controller}[^/]+)\\.({action}[^/]+)"),
		actionArgs: []string { "controller", "action" },
	},
}

// Run the test cases above.
func TestComputeRoute(t *testing.T) {
	for routeLine, expected := range routeTestCases {
		method, path, action, found := parseRouteLine(routeLine)
		if ! found {
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

// const TEST_ROUTES = `
// # This is a comment
// GET  /                       Application.index
// GET  /app/{id}               Application.show
// POST /app/{id}               Application.save

// GET  /public/                staticDir:public
// *    /_{controller}/{action} {controller}.{action}
// `

// func (t *testing.T) TestRouter {
// 	router := NewRouter(TEST_ROUTES)
// }

// Helpers

func eq(t *testing.T, name string, a, b interface{}) {
	if a != b {
		t.Error(name, ": (actual)", a, " != ", b, "(expected)")
	}
}

