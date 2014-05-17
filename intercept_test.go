package revel

import (
	"reflect"
	"testing"
)

var funcP = func(c *Controller) Result { return nil }
var funcP2 = func(c *Controller) Result { return nil }

type InterceptController struct{ *Controller }
type InterceptControllerN struct{ InterceptController }
type InterceptControllerP struct{ *InterceptController }
type InterceptControllerNP struct {
	*Controller
	InterceptControllerN
	InterceptControllerP
}

func (c InterceptController) methN() Result  { return nil }
func (c *InterceptController) methP() Result { return nil }

func (c InterceptControllerN) methNN() Result  { return nil }
func (c *InterceptControllerN) methNP() Result { return nil }
func (c InterceptControllerP) methPN() Result  { return nil }
func (c *InterceptControllerP) methPP() Result { return nil }

// Methods accessible from InterceptControllerN
var METHODS_N = []interface{}{
	InterceptController.methN,
	(*InterceptController).methP,
	InterceptControllerN.methNN,
	(*InterceptControllerN).methNP,
}

// Methods accessible from InterceptControllerP
var METHODS_P = []interface{}{
	InterceptController.methN,
	(*InterceptController).methP,
	InterceptControllerP.methPN,
	(*InterceptControllerP).methPP,
}

// This checks that all the various kinds of interceptor functions/methods are
// properly invoked.
func TestInvokeArgType(t *testing.T) {
	n := InterceptControllerN{InterceptController{&Controller{}}}
	p := InterceptControllerP{&InterceptController{&Controller{}}}
	np := InterceptControllerNP{&Controller{}, n, p}
	testInterceptorController(t, reflect.ValueOf(&n), METHODS_N)
	testInterceptorController(t, reflect.ValueOf(&p), METHODS_P)
	testInterceptorController(t, reflect.ValueOf(&np), METHODS_N)
	testInterceptorController(t, reflect.ValueOf(&np), METHODS_P)
}

func testInterceptorController(t *testing.T, appControllerPtr reflect.Value, methods []interface{}) {
	interceptors = []*Interception{}
	InterceptFunc(funcP, BEFORE, appControllerPtr.Elem().Interface())
	InterceptFunc(funcP2, BEFORE, ALL_CONTROLLERS)
	for _, m := range methods {
		InterceptMethod(m, BEFORE)
	}
	ints := getInterceptors(BEFORE, appControllerPtr)

	if len(ints) != 6 {
		t.Fatalf("N: Expected 6 interceptors, got %d.", len(ints))
	}

	testInterception(t, ints[0], reflect.ValueOf(&Controller{}))
	testInterception(t, ints[1], reflect.ValueOf(&Controller{}))
	for i := range methods {
		testInterception(t, ints[i+2], appControllerPtr)
	}
}

func testInterception(t *testing.T, intc *Interception, arg reflect.Value) {
	val := intc.Invoke(arg)
	if !val.IsNil() {
		t.Errorf("Failed (%s): Expected nil got %v", intc, val)
	}
}
