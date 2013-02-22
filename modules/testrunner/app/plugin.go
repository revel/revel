package app

import (
	"fmt"
	"github.com/robfig/revel"
)

type TestRunnerPlugin struct {
	revel.EmptyPlugin
}

func (t TestRunnerPlugin) OnRoutesLoaded(router *revel.Router) {
	router.Routes = append([]*revel.Route{
		revel.NewRoute("GET", "/@tests", "TestRunner.Index", ""),
		revel.NewRoute("GET", "/@tests.list", "TestRunner.List", ""),
		revel.NewRoute("GET", "/@tests/public/", "staticDir:testrunner:public", ""),
		revel.NewRoute("GET", "/@tests/{suite}/{test}", "TestRunner.Run", ""),
	}, router.Routes...)
	fmt.Println("Go to /@tests to run the tests.")
}
