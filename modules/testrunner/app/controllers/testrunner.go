package controllers

import (
	"fmt"
	"github.com/robfig/revel"
	"github.com/robfig/revel/modules/testrunner/app"
	"reflect"
	"runtime/debug"
)

type TestRunner struct {
	*rev.Controller
}

type TestSuiteDesc struct {
	Name  string
	Tests []TestDesc
}

type TestDesc struct {
	Name string
	// TODO: Add comment as description here.
}

type TestResult struct {
	Success      bool
	ErrorMessage string
	Stack        string
}

func (c TestRunner) Index() rev.Result {
	var functionalTestSuites []TestSuiteDesc
	for _, testSuite := range rev.FunctionalTests {
		functionalTestSuites = append(functionalTestSuites, DescribeSuite(testSuite))
	}
	return c.Render(functionalTestSuites)
}

// Run runs a single test, given by the argument. 
func (c TestRunner) Run(suite, test string) rev.Result {
	var result TestResult
	for _, testSuite := range rev.FunctionalTests {
		t := reflect.TypeOf(testSuite).Elem()
		if t.Name() == suite {
			v := reflect.New(t)
			func() {
				defer func() {
					if err := recover(); err != nil {
						result.ErrorMessage = fmt.Sprint(err)
						result.Stack = string(debug.Stack())
					}
				}()
				v.MethodByName(test).Call([]reflect.Value{})
				result.Success = true
			}()
			break
		}
	}
	return c.RenderJson(result)
}

func DescribeSuite(testSuite interface{}) TestSuiteDesc {
	t := reflect.TypeOf(testSuite).Elem()

	// Get a list of methods of the embedded test type.
	super := t.Field(0).Type
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
		if mt.NumIn() == 1 && mt.NumOut() == 0 && mt.In(0) == t && !isSuperMethod {
			tests = append(tests, TestDesc{m.Name})
		}
	}

	return TestSuiteDesc{
		Name:  t.Name(),
		Tests: tests,
	}
}

func init() {
	rev.RegisterPlugin(app.TestRunnerPlugin{})
}
