package controllers

import (
	"bytes"
	"fmt"
	"github.com/revel/revel"
	"html"
	"html/template"
	"reflect"
	"strings"
)

type TestRunner struct {
	*revel.Controller
}

type TestSuiteDesc struct {
	Name  string
	Tests []TestDesc
}

type TestDesc struct {
	Name string
}

type TestSuiteResult struct {
	Name    string
	Passed  bool
	Results []TestResult
}

type TestResult struct {
	Name         string
	Passed       bool
	ErrorHtml    template.HTML
	ErrorSummary string
}

var NONE = []reflect.Value{}

func (c TestRunner) Index() revel.Result {
	var testSuites []TestSuiteDesc
	for _, testSuite := range revel.TestSuites {
		testSuites = append(testSuites, DescribeSuite(testSuite))
	}
	return c.Render(testSuites)
}

// Run runs a single test, given by the argument.
func (c TestRunner) Run(suite, test string) revel.Result {
	result := TestResult{Name: test}
	for _, testSuite := range revel.TestSuites {
		t := reflect.TypeOf(testSuite).Elem()
		if t.Name() != suite {
			continue
		}

		// Found the suite, create a new instance and run the named method.
		v := reflect.New(t)
		func() {
			defer func() {
				if err := recover(); err != nil {
					error := revel.NewErrorFromPanic(err)
					if error == nil {
						result.ErrorHtml = template.HTML(html.EscapeString(fmt.Sprint(err)))
					} else {
						var buffer bytes.Buffer
						tmpl, _ := revel.MainTemplateLoader.Template("TestRunner/FailureDetail.html")
						tmpl.Render(&buffer, error)
						result.ErrorSummary = errorSummary(error)
						result.ErrorHtml = template.HTML(buffer.String())
					}
				}
			}()

			// Initialize the test suite with a NewTestSuite()
			testSuiteInstance := v.Elem().FieldByName("TestSuite")
			testSuiteInstance.Set(reflect.ValueOf(revel.NewTestSuite()))

			// Call Before(), call the test, and call After().
			if m := v.MethodByName("Before"); m.IsValid() {
				m.Call(NONE)
			}

			if m := v.MethodByName("After"); m.IsValid() {
				defer m.Call(NONE)
			}

			v.MethodByName(test).Call(NONE)

			// No panic means success.
			result.Passed = true
		}()
		break
	}
	return c.RenderJson(result)
}

// List returns a JSON list of test suites and tests.
// Used by the "test" command line tool.
func (c TestRunner) List() revel.Result {
	var testSuites []TestSuiteDesc
	for _, testSuite := range revel.TestSuites {
		testSuites = append(testSuites, DescribeSuite(testSuite))
	}
	return c.RenderJson(testSuites)
}

func DescribeSuite(testSuite interface{}) TestSuiteDesc {
	t := reflect.TypeOf(testSuite)

	// Get a list of methods of the embedded test type.
	super := t.Elem().Field(0).Type
	superMethodNameSet := map[string]struct{}{}
	for i := 0; i < super.NumMethod(); i++ {
		superMethodNameSet[super.Method(i).Name] = struct{}{}
	}

	// Get a list of methods on the test suite that take no parameters, return
	// no results, and were not part of the embedded type's method set.
	var tests []TestDesc
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		mt := m.Type
		_, isSuperMethod := superMethodNameSet[m.Name]
		if mt.NumIn() == 1 &&
			mt.NumOut() == 0 &&
			mt.In(0) == t &&
			!isSuperMethod &&
			strings.HasPrefix(m.Name, "Test") {
			tests = append(tests, TestDesc{m.Name})
		}
	}

	return TestSuiteDesc{
		Name:  t.Elem().Name(),
		Tests: tests,
	}
}

func errorSummary(error *revel.Error) string {
	var message = fmt.Sprintf("%4sStatus: %s\n%4sIn %s", "", error.Description, "", error.Path)
	if error.Line != 0 {
		message += fmt.Sprintf(" (around line %d): ", error.Line)
		for _, line := range error.ContextSource() {
			if line.IsError {
				message += line.Source
			}
		}
	}
	return message
}
