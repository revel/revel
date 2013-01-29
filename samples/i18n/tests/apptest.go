package tests

import "github.com/robfig/revel"

type ApplicationTest struct {
	revel.TestSuite
}

func (t ApplicationTest) Before() {
	println("Set up")
}

func (t ApplicationTest) TestThatIndexPageWorks() {
	t.Get("/")
	t.AssertOk()
	t.AssertContentType("text/html")
}

func (t ApplicationTest) After() {
	println("Tear down")
}
