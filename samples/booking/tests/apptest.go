package tests

import "github.com/robfig/revel"

type ApplicationTest struct {
	rev.FunctionalTest
}

func (t ApplicationTest) TestThatIndexPageWorks() {
	t.GetPath("/")
	t.AssertOk()
	t.AssertContentType("text/html")
}
