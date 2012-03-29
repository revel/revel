package play

import (
	"log"
	"reflect"
)

// An "interceptor" is a function that is invoked by the framework at a
// designated time (BEFORE or AFTER) an action invcation.
//
// Since an interceptor may be used across many user Controllers, it is a
// function that takes the base Controller, rather than a method on a user
// controller.
//
// An interceptor may optionally return a Result (instead of nil).  Depending on
// when the interceptor was invoked, the response is different:
// 1. BEFORE:  No further interceptors are invoked, and neither is the action.
// 2. AFTER: Further interceptors are still run.
// In all cases, any returned Result will take the place of any existing Result.
// But in the BEFORE case, that returned Result is guaranteed to be final, while
// in the AFTER case it is possible that a further interceptor could emit its
// own Result.
//
// Interceptors are called in the order that they are added.
//
// ***
//
// Two types of interceptors are provided: Funcs and Methods
//
// Func Interceptors may apply to any / all Controllers.
//
//   func example(*play.Controller) play.Result
//
// Method Interceptors are provided so that properties can be set on application
// controllers.
//
//   func (c AppController) example() play.Result
//   func (c *AppController) example() play.Result
//
type InterceptorFunc func(*Controller) Result
type InterceptorMethod interface{}
type InterceptTime int

const (
	BEFORE InterceptTime = iota
	AFTER
)

type Interception struct {
	When InterceptTime

	function InterceptorFunc
	method   InterceptorMethod

	callableValue reflect.Value
	targetType    reflect.Type
}

// Perform the given interception.
// val is a pointer to the App Controller.
func (i Interception) Invoke(val reflect.Value) reflect.Value {
	// Figure out what type of parameter the interceptor needs.
	var arg reflect.Value
	switch i.callableValue.Type().In(0) {
	case reflect.TypeOf(Controller{}):
		arg = val.Elem().Field(0).Elem()
	case reflect.TypeOf(&Controller{}):
		arg = val.Elem().Field(0)
	case val.Type():
		arg = val
	case val.Type().Elem():
		arg = val.Elem()
	default:
		// This should never happen.
		// Validation happens at the time the interceptor is installed.
		log.Fatalln("Unrecognized parameter type.")
	}
	vals := i.callableValue.Call([]reflect.Value{arg})
	return vals[0]
}

var interceptors []*Interception

// Install a general interceptor.
// This can be applied to any Controller.
// It must have the signature of:
// func example(c *play.Controller) play.Result
func InterceptFunc(intc InterceptorFunc, when InterceptTime, target interface{}) {
	interceptors = append(interceptors, &Interception{
		When:          when,
		function:      intc,
		callableValue: reflect.ValueOf(intc),
		targetType:    reflect.TypeOf(target),
	})
}

// Install an interceptor method that applies to its own Controller.
// func (c AppController) example() play.Result
// func (c *AppController) example() play.Result
func InterceptMethod(intc InterceptorMethod, when InterceptTime) {
	methodType := reflect.TypeOf(intc)
	if methodType.Kind() != reflect.Func || methodType.NumOut() != 1 || methodType.NumIn() != 1 {
		log.Fatalln("Interceptor method should have signature like",
			"'func (c *AppController) example() play.Result' but was", methodType)
	}
	interceptors = append(interceptors, &Interception{
		When:          when,
		method:        intc,
		callableValue: reflect.ValueOf(intc),
		targetType:    methodType.In(0),
	})
}

func getInterceptors(when InterceptTime, targetType reflect.Type) []*Interception {
	result := []*Interception{}
	for _, intc := range interceptors {
		if intc.When == when &&
			(intc.targetType == targetType || intc.targetType == targetType.Elem()) {
			result = append(result, intc)
		}
	}
	return result
}
