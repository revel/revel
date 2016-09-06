package revel

import (
	"reflect"
	"testing"
)

var funcP = func(c *Controller) Result {
	return nil
}
var funcP2 = func(c *Controller) Result {
	return nil
}
var func123 = func(c *Controller) Result {
	return RenderTextResult{text: "123"}
}
var func456 = func(c *Controller) Result {
	return RenderTextResult{text: "456"}
}

type InterceptController struct{ *Controller }
type InterceptControllerN struct{ InterceptController }
type InterceptControllerP struct{ *InterceptController }
type InterceptControllerNP struct {
	*Controller
	InterceptControllerN
	InterceptControllerP
}

func (c InterceptController) methN() Result {
	return nil
}
func (c *InterceptController) methP() Result {
	return nil
}

func (c InterceptControllerN) methNN() Result {
	return nil
}
func (c *InterceptControllerN) methNP() Result {
	return nil
}
func (c InterceptControllerP) methPN() Result {
	return nil
}
func (c *InterceptControllerP) methPP() Result {
	return nil
}

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

func TestOrderedInterceptors(t *testing.T) {
	interceptors = SortInterceptions{}
	controller := InterceptController{}
	controllerValue := reflect.ValueOf(controller)
	InterceptFunc(func456, BEFORE, ALL_CONTROLLERS, 2)
	InterceptFunc(func123, BEFORE, ALL_CONTROLLERS, 1)
	ints := getInterceptors(BEFORE, controllerValue)
	if len(ints) != 2 {
		t.Fatalf("N: Expected 2, got %d", len(ints))
	}
	testFuncReturnRenderTextResult(t, ints[0].function, "123")
	testFuncReturnRenderTextResult(t, ints[1].function, "456")
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

func testFuncReturnRenderTextResult(t *testing.T, f InterceptorFunc, text string) {
	if f == nil {
		t.Fatalf("Nil function (Should return RenderTextResult with text: %s)", text)
	}
	res := f(&Controller{})
	r, ok := res.(RenderTextResult)
	if !ok {
		t.Fatalf("Expected return type RenderTextResult, got %T", res)
	}
	if r.text != text {
		t.Fatalf("Expected text value = '%s', got '%s'", text, r.text)
	}
}
