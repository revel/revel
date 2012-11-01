package app

import (
	"fmt"
	"github.com/robfig/revel"
)

type TestRunnerPlugin struct {
	rev.EmptyPlugin
}

func (t TestRunnerPlugin) OnRoutesLoaded(router *rev.Router) {
	router.Routes = append([]*rev.Route{
		rev.NewRoute("GET", "/@tests", "TestRunner.Index"),
		rev.NewRoute("GET", "/@tests.list", "TestRunner.List"),
		rev.NewRoute("GET", "/@tests/public/", "staticDir:testrunner:public"),
		rev.NewRoute("GET", "/@tests/{suite}", "TestRunner.Run"),
		rev.NewRoute("GET", "/@tests/{suite}/{test}", "TestRunner.Run"),
		rev.NewRoute("POST", "/@tests/{<.*>test}", "TestRunner.SaveResult"),
		rev.NewRoute("GET", "/@tests/emails", "TestRunner.MockEmail"),
		rev.NewRoute("GET", "/@tests/cache", "TestRunner.CacheEntry"),
	}, router.Routes...)
	fmt.Println("Go to /@tests to run the tests.")
}
