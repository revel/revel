package revel

import (
	"reflect"
	"strings"
	"testing"
)

type Plurals struct { //test plural resource

}

type Singular struct { //test singular resource

}

type EmptyPlurals struct { //test empty plural resource

}

type EmptySingular struct { //test empty singular resource

}

//containsRoute() is a utility function used to make sure the resulting slice of routes contains each of the routes that it is supposed to
func containsRoute(routes []*Route, contained *Route) bool {
	for _, route := range routes {
		if route.Method != contained.Method {
			continue
		}
		if route.Path != contained.Path {
			continue
		}
		if route.Action != contained.Action {
			continue
		}
		if route.ControllerName != contained.ControllerName {
			continue
		}
		if route.MethodName != contained.MethodName {
			continue
		}
		if len(route.FixedParams) != len(contained.FixedParams) {
			continue
		}
		return true
	}
	return false
}

/*
	TestPluralResource() makes sure that given a controller class with a pluralized name and the appropriate methods,
	the Router should generate the following RESTful routes:
	Index,New,Create,Show,Edit,Update,Destroy
*/
func TestPluralResource(t *testing.T) {
	correctRoutes := []*Route{
		&Route{
			Method:         "GET",
			Path:           "/plurals",
			Action:         "Plurals.Index",
			ControllerName: "Plurals",
			MethodName:     "Index",
		},
		&Route{
			Method:         "GET",
			Path:           "/plurals/new",
			Action:         "Plurals.New",
			ControllerName: "Plurals",
			MethodName:     "New",
		},
		&Route{
			Method:         "POST",
			Path:           "/plurals",
			Action:         "Plurals.Create",
			ControllerName: "Plurals",
			MethodName:     "Create",
		},
		&Route{
			Method:         "GET",
			Path:           "/plurals/:plural_id",
			Action:         "Plurals.Show",
			ControllerName: "Plurals",
			MethodName:     "Show",
		},
		&Route{
			Method:         "GET",
			Path:           "/plurals/:plural_id/edit",
			Action:         "Plurals.Edit",
			ControllerName: "Plurals",
			MethodName:     "Edit",
		},
		&Route{
			Method:         "PUT",
			Path:           "/plurals/:plural_id",
			Action:         "Plurals.Update",
			ControllerName: "Plurals",
			MethodName:     "Update",
		},
		&Route{
			Method:         "DELETE",
			Path:           "/plurals/:plural_id",
			Action:         "Plurals.Destroy",
			ControllerName: "Plurals",
			MethodName:     "Destroy",
		},
	}

	RegisterController((*Plurals)(nil),
		[]*MethodType{
			&MethodType{
				Name: "Index",
			},
			&MethodType{
				Name: "New",
			},
			&MethodType{
				Name: "Create",
			},
			&MethodType{
				Name: "Show",
				Args: []*MethodArg{
					{"monkey_id", reflect.TypeOf((*int)(nil))},
				},
			},
			&MethodType{
				Name: "Edit",
				Args: []*MethodArg{
					{"monkey_id", reflect.TypeOf((*int)(nil))},
				},
			},
			&MethodType{
				Name: "Update",
				Args: []*MethodArg{
					{"monkey_id", reflect.TypeOf((*int)(nil))},
				},
			},
			&MethodType{
				Name: "Destroy",
				Args: []*MethodArg{
					{"monkey_id", reflect.TypeOf((*int)(nil))},
				},
			},
		})

	//Forward Routing
	routes, err := parseRoutes("", "RESOURCE /plurals Plurals", true)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(routes) > 7 {
		t.Error("Found too many routes")
	}

	for _, route := range correctRoutes {
		if !containsRoute(routes, route) {
			t.Error("Missing route: " + route.MethodName)
		}
	}

	//Reverse Routing
	MainRouter = NewRouter("")
	MainRouter.Routes = routes
	MainRouter.updateTree()

	for _, route := range correctRoutes {
		args := map[string]string{}
		if strings.Contains(route.Path, ":plural_id") {
			args["plural_id"] = ":plural_id"
		}
		path := MainRouter.Reverse(route.Action, args)
		if path.Url != route.Path || path.Method != route.Method {
			t.Errorf("Reverse Routing, got %s %s instead of %s %s", path.Method, path.Url, route.Method, route.Path)
		}
	}
}

/*
	TestSingularResource() makes sure that given a controller class with a singularized name and the appropriate methods,
	the Router should generate the following RESTful routes:
	New,Create,Show,Edit,Update,Destroy
*/
func TestSingularResource(t *testing.T) {
	correctRoutes := []*Route{
		&Route{
			Method:         "GET",
			Path:           "/singular/new",
			Action:         "Singular.New",
			ControllerName: "Singular",
			MethodName:     "New",
		},
		&Route{
			Method:         "POST",
			Path:           "/singular",
			Action:         "Singular.Create",
			ControllerName: "Singular",
			MethodName:     "Create",
		},
		&Route{
			Method:         "GET",
			Path:           "/singular",
			Action:         "Singular.Show",
			ControllerName: "Singular",
			MethodName:     "Show",
		},
		&Route{
			Method:         "GET",
			Path:           "/singular/edit",
			Action:         "Singular.Edit",
			ControllerName: "Singular",
			MethodName:     "Edit",
		},
		&Route{
			Method:         "PUT",
			Path:           "/singular",
			Action:         "Singular.Update",
			ControllerName: "Singular",
			MethodName:     "Update",
		},
		&Route{
			Method:         "DELETE",
			Path:           "/singular",
			Action:         "Singular.Destroy",
			ControllerName: "Singular",
			MethodName:     "Destroy",
		},
	}

	RegisterController((*Singular)(nil),
		[]*MethodType{
			&MethodType{
				Name: "New",
			},
			&MethodType{
				Name: "Create",
			},
			&MethodType{
				Name: "Show",
			},
			&MethodType{
				Name: "Edit",
			},
			&MethodType{
				Name: "Update",
			},
			&MethodType{
				Name: "Destroy",
			},
		})

	//Forward Routing
	routes, err := parseRoutes("", "RESOURCE /singular Singular", true)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(routes) > 6 {
		t.Error("Found too many routes")
	}

	for _, route := range correctRoutes {
		if !containsRoute(routes, route) {
			t.Error("Missing route: " + route.MethodName)
		}
	}

	//Reverse Routing
	MainRouter = NewRouter("")
	MainRouter.Routes = routes
	MainRouter.updateTree()

	for _, route := range correctRoutes {
		path := MainRouter.Reverse(route.Action, nil)
		if path.Url != route.Path || path.Method != route.Method {
			t.Errorf("Reverse Routing, got %s %s instead of %s %s", path.Method, path.Url, route.Method, route.Path)
		}
	}
}

/*
	TestEmptyPluralResource() makes sure that given a controller class with a pluralized name but without the appropriate methods,
	the Router should not generate any routes:
	The Router should not generate routes for actions that don't exist
*/
func TestEmptyPluralResource(t *testing.T) {
	RegisterController((*EmptyPlurals)(nil), []*MethodType{})
	routes, err := parseRoutes("", "RESOURCE /emptyplurals EmptyPlurals", true)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(routes) > 0 {
		t.Error("Found too many routes")
	}
}

/*
	TestEmptySingularResource() makes sure that given a controller class with a singularized name but without the appropriate methods,
	the Router should not generate any routes:
	The Router should not generate routes for actions that don't exist
*/
func TestEmptySingularResource(t *testing.T) {
	RegisterController((*EmptySingular)(nil), []*MethodType{})
	routes, err := parseRoutes("", "RESOURCE /emptysingular EmptySingular", true)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(routes) > 0 {
		t.Error("Found too many routes")
	}
}
