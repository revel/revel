package rev

import (
	"fmt"
	"reflect"
	"testing"
)

// These tests verify that Controllers are initialized properly, given the range
// of embedding possibilities..

type N struct{ Controller }
type P struct{ *Controller }

type NN struct{ N }
type NP struct{ *N }
type PN struct{ P }
type PP struct{ *P }

type NNN struct{ NN }
type NPN struct{ NP }
type PNP struct{ *PN }
type PPP struct{ *PP }

var GENERATIONS = [][]interface{}{
	{N{}, P{}},
	{NN{}, NP{}, PN{}, PP{}},
	{NNN{}, NPN{}, PNP{}, PPP{}},
}

// This test constructs a bunch of hypothetical app controllers, and verifies
// that the embedded Controller field was set correctly.
func TestNewAppController(t *testing.T) {
	controller := &Controller{Name: "Test"}
	for gen, structs := range GENERATIONS {
		for _, st := range structs {
			typ := reflect.TypeOf(st)
			t.Log("Type: " + typ.String())
			fmt.Println("Type: " + typ.String())
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
	if val.Interface().(*PNP).PN.P.Controller.Name != "Test" {
		t.Error("PNP not initialized.")
	}
}
