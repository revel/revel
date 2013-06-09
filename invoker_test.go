package revel

import (
	"net/url"
	"reflect"
	"testing"
)

// These tests verify that Controllers are initialized properly, given the range
// of embedding possibilities..

type P struct{ *Controller }

type PN struct{ P }

type PNN struct{ PN }

// Embedded via two paths
type P2 struct{ *Controller }
type PP2 struct {
	*Controller // Need to embed this explicitly to avoid duplicate selector.
	P
	P2
	PNN
}

var GENERATIONS = []interface{}{P{}, PN{}, PNN{}}

func TestFindControllers(t *testing.T) {
	controllers = make(map[string]*ControllerType)
	RegisterController((*P)(nil), nil)
	RegisterController((*PN)(nil), nil)
	RegisterController((*PNN)(nil), nil)
	RegisterController((*PP2)(nil), nil)

	// Test construction of indexes to each *Controller
	checkSearchResults(t, P{}, [][]int{{0}})
	checkSearchResults(t, PN{}, [][]int{{0, 0}})
	checkSearchResults(t, PNN{}, [][]int{{0, 0, 0}})
	checkSearchResults(t, PP2{}, [][]int{{0}, {1, 0}, {2, 0}, {3, 0, 0, 0}})
}

func checkSearchResults(t *testing.T, obj interface{}, expected [][]int) {
	actual := findControllers(reflect.TypeOf(obj))
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Indexes do not match.  expected %v actual %v", expected, actual)
	}
}

func TestSetAction(t *testing.T) {
	controllers = make(map[string]*ControllerType)
	RegisterController((*P)(nil), []*MethodType{{Name: "Method"}})
	RegisterController((*PNN)(nil), []*MethodType{{Name: "Method"}})
	RegisterController((*PP2)(nil), []*MethodType{{Name: "Method"}})

	// Test that all *revel.Controllers are initialized.
	c := &Controller{Name: "Test"}
	if err := c.SetAction("P", "Method"); err != nil {
		t.Error(err)
	} else if c.AppController.(*P).Controller != c {
		t.Errorf("P not initialized")
	}

	if err := c.SetAction("PNN", "Method"); err != nil {
		t.Error(err)
	} else if c.AppController.(*PNN).Controller != c {
		t.Errorf("PNN not initialized")
	}

	// PP2 has 4 different slots for *Controller.
	if err := c.SetAction("PP2", "Method"); err != nil {
		t.Error(err)
	} else if pp2 := c.AppController.(*PP2); pp2.Controller != c ||
		pp2.P.Controller != c ||
		pp2.P2.Controller != c ||
		pp2.PNN.Controller != c {
		t.Errorf("PP2 not initialized")
	}
}

func BenchmarkSetAction(b *testing.B) {
	type Mixin1 struct {
		*Controller
		x, y int
		foo  string
	}
	type Mixin2 struct {
		*Controller
		a, b float64
		bar  string
	}

	type Benchmark struct {
		*Controller
		Mixin1
		Mixin2
		user interface{}
		guy  string
	}

	RegisterController((*Mixin1)(nil), []*MethodType{{Name: "Method"}})
	RegisterController((*Mixin2)(nil), []*MethodType{{Name: "Method"}})
	RegisterController((*Benchmark)(nil), []*MethodType{{Name: "Method"}})
	c := Controller{
		RenderArgs: make(map[string]interface{}),
	}

	for i := 0; i < b.N; i++ {
		if err := c.SetAction("Benchmark", "Method"); err != nil {
			b.Errorf("Failed to set action: %s", err)
			return
		}
	}
}

func BenchmarkInvoker(b *testing.B) {
	startFakeBookingApp()
	c := Controller{
		RenderArgs: make(map[string]interface{}),
	}
	if err := c.SetAction("Hotels", "Show"); err != nil {
		b.Errorf("Failed to set action: %s", err)
		return
	}
	c.Request = NewRequest(showRequest)
	c.Params = &Params{Values: make(url.Values)}
	c.Params.Set("id", "3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ActionInvoker(&c, nil)
	}
}
