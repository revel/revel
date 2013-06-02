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
type PP struct{ *P }

type PNN struct{ PN }
type PPN struct{ PP }
type PNP struct{ *PN }
type PPP struct{ *PP }

// Embedded via two paths
type P2 struct{ *Controller }
type PP2 struct {
	*Controller // Need to embed this explicitly to avoid duplicate selector.
	P
	P2
}

var GENERATIONS = [][]interface{}{
	{P{}},
	{PN{}, PP{}},
	{PNN{}, PPN{}, PNP{}, PPP{}},
}

// This test constructs a bunch of hypothetical app controllers, and verifies
// that the embedded Controller field was set correctly.
func TestNewAppController(t *testing.T) {
	controller := &Controller{Name: "Test"}
	for gen, structs := range GENERATIONS {
		for _, st := range structs {
			typ := reflect.TypeOf(st)
			val := initNewAppController(typ, controller)

			// Drill into the embedded fields to get to the Controller.
			for i := 0; i < gen+1; i++ {
				if val.Kind() == reflect.Ptr {
					val = val.Elem()
				}
				val = val.Field(0)
			}

			var name string
			if val.Type().Kind() == reflect.Ptr {
				name = val.Interface().(*Controller).Name
			} else {
				name = val.Interface().(Controller).Name
			}

			if name != "Test" {
				t.Error("Fail: " + typ.String())
			}
		}
	}
}

// Since the test machinery that goes through all the structs is non-trivial,
// have one redundant test that covers just one complicated case but is dead
// simple.
func TestNewAppController2(t *testing.T) {
	val := initNewAppController(reflect.TypeOf(PNP{}), &Controller{Name: "Test"})
	pnp := val.Interface().(*PNP)
	if pnp.PN.P.Controller.Name != "Test" {
		t.Error("PNP not initialized.")
	}
	if pnp.Controller.Name != "Test" {
		t.Error("PNP promotion not working.")
	}
}

func TestMultiEmbedding(t *testing.T) {
	val := initNewAppController(reflect.TypeOf(PP2{}), &Controller{Name: "Test"})
	pp2 := val.Interface().(*PP2)
	if pp2.P.Controller.Name != "Test" {
		t.Error("P not initialized.")
	}

	if pp2.P2.Controller.Name != "Test" {
		t.Error("P2 not initialized.")
	}

	if pp2.Controller.Name != "Test" {
		t.Error("PP2 promotion not working.")
	}

	if pp2.P.Controller != pp2.P2.Controller || pp2.Controller != pp2.P.Controller {
		t.Error("Controllers not pointing to the same thing.")
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
