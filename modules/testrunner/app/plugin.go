package app

import (
	"fmt"
	"github.com/robfig/revel"
)

type TestRunnerPlugin struct {
	revel.EmptyPlugin
}

func (t TestRunnerPlugin) OnAppStart() {
	fmt.Println("Go to /@tests to run the tests.")
}
