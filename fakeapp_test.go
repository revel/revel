// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"os"
	"path/filepath"
	"reflect"
)

type Hotel struct {
	HotelID          int
	Name, Address    string
	City, State, Zip string
	Country          string
	Price            int
}

type Hotels struct {
	*Controller
}

type Static struct {
	*Controller
}

type Implicit struct {
	*Controller
}

type Application struct {
	*Controller
}

func (c Hotels) Show(id int) Result {
	title := "View Hotel"
	hotel := &Hotel{id, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300}
	// The line number below must match the one with the code : RenderArgNames: map[int][]string{43: {"title", "hotel"}},
	return c.Render(title, hotel)
}

func (c Hotels) Book(id int) Result {
	hotel := &Hotel{id, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300}
	return c.RenderJSON(hotel)
}

func (c Hotels) Index() Result {
	return c.RenderText("Hello, World!")
}

func (c Static) Serve(prefix, path string) Result {
	var basePath, dirName string

	if !filepath.IsAbs(dirName) {
		basePath = BasePath
	}

	fname := filepath.Join(basePath, prefix, path)
	file, err := os.Open(fname)
	if os.IsNotExist(err) {
		return c.NotFound("")
	} else if err != nil {
		RevelLog.Errorf("Problem opening file (%s): %s ", fname, err)
		return c.NotFound("This was found but not sure why we couldn't open it.")
	}
	return c.RenderFile(file, "")
}

// Register controllers is in its own function so the route test can use it as well
func registerControllers() {
	controllers = make(map[string]*ControllerType)
	RaiseEvent(ROUTE_REFRESH_REQUESTED, nil)
	RegisterController((*Hotels)(nil),
		[]*MethodType{
			{
				Name: "Index",
			},
			{
				Name: "Show",
				Args: []*MethodArg{
					{"id", reflect.TypeOf((*int)(nil))},
				},
				RenderArgNames: map[int][]string{41: {"title", "hotel"}},
			},
			{
				Name: "Book",
				Args: []*MethodArg{
					{"id", reflect.TypeOf((*int)(nil))},
				},
			},
		})

	RegisterController((*Static)(nil),
		[]*MethodType{
			{
				Name: "Serve",
				Args: []*MethodArg{
					{Name: "prefix", Type: reflect.TypeOf((*string)(nil))},
					{Name: "filepath", Type: reflect.TypeOf((*string)(nil))},
				},
				RenderArgNames: map[int][]string{},
			},
		})
	RegisterController((*Implicit)(nil),
		[]*MethodType{
			{
				Name: "Implicit",
				Args: []*MethodArg{
					{Name: "prefix", Type: reflect.TypeOf((*string)(nil))},
					{Name: "filepath", Type: reflect.TypeOf((*string)(nil))},
				},
				RenderArgNames: map[int][]string{},
			},
		})
	RegisterController((*Application)(nil),
		[]*MethodType{
			{
				Name: "Application",
				Args: []*MethodArg{
					{Name: "prefix", Type: reflect.TypeOf((*string)(nil))},
					{Name: "filepath", Type: reflect.TypeOf((*string)(nil))},
				},
				RenderArgNames: map[int][]string{},
			},
			{
				Name: "Index",
				Args: []*MethodArg{
					{Name: "foo", Type: reflect.TypeOf((*string)(nil))},
					{Name: "bar", Type: reflect.TypeOf((*string)(nil))},
				},
				RenderArgNames: map[int][]string{},
			},
		})
}
func startFakeBookingApp() {
	Init("prod", "github.com/revel/revel/testdata", "")

	MainTemplateLoader = NewTemplateLoader([]string{ViewsPath, filepath.Join(RevelPath, "templates")})
	if err := MainTemplateLoader.Refresh(); err != nil {
		RevelLog.Fatal("Template error","error",err)
	}

	registerControllers()

	InitServerEngine(9000, GO_NATIVE_SERVER_ENGINE)
	RaiseEvent(ENGINE_BEFORE_INITIALIZED, nil)
	InitServer()

	RaiseEvent(ENGINE_STARTED, nil)
}
