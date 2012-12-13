package rev

import (
	"log"
	"reflect"
)

// An "interceptor" is functionality invoked by the framework BEFORE or AFTER
// an action.
//
// An interceptor may optionally return a Result (instead of nil).  Depending on
// when the interceptor was invoked, the response is different:
// 1. BEFORE:  No further interceptors are invoked, and neither is the action.
// 2. AFTER: Further interceptors are still run.
// In all cases, any returned Result will take the place of any existing Result.
//
// In the BEFORE case, that returned Result is guaranteed to be final, while
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
//   func example(*rev.Controller) rev.Result
//
// Method Interceptors are provided so that properties can be set on application
// controllers.
//
//   func (c AppController) example() rev.Result
//   func (c *AppController) example() rev.Result
//
type InterceptorFunc func(*Controller) Result
type InterceptorMethod interface{}
type InterceptTime int

const (
	BEFORE InterceptTime = iota
	AFTER
	PANIC
	FINALLY
)

type InterceptTarget int

const (
	ALL_CONTROLLERS InterceptTarget = iota
)

type Interception struct {
	When InterceptTime

	function InterceptorFunc
	method   InterceptorMethod

	callable     reflect.Value
	target       reflect.Type
	interceptAll bool
}

// Perform the given interception.
// val is a pointer to the App Controller.
func (i Interception) Invoke(val reflect.Value) reflect.Value {
	var arg reflect.Value
	if i.function == nil {
		// If it's an InterceptorMethod, then we have to pass in the target type.
		arg = findTarget(val, i.target)
	} else {
		// If it's an InterceptorFunc, then the type must be *Controller.
		// We can find that by following the embedded types up the chain.
		for val.Type() != controllerPtrType {
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			val = val.Field(0)
		}
		arg = val
	}

	vals := i.callable.Call([]reflect.Value{arg})
	return vals[0]
}

type InterceptorPlugin struct {
	EmptyPlugin
}

func (p InterceptorPlugin) BeforeRequest(c *Controller) {
	invokeInterceptors(BEFORE, c)
}

func (p InterceptorPlugin) AfterRequest(c *Controller) {
	invokeInterceptors(AFTER, c)
}

func (p InterceptorPlugin) OnException(c *Controller, err interface{}) {
	invokeInterceptors(PANIC, c)
}

func (p InterceptorPlugin) Finally(c *Controller) {
	invokeInterceptors(FINALLY, c)
}

func invokeInterceptors(when InterceptTime, c *Controller) {
	appControllerPtr := reflect.ValueOf(c.AppController)
	result := func() Result {
		var result Result
		for _, intc := range getInterceptors(when, appControllerPtr) {
			resultValue := intc.Invoke(appControllerPtr)
			if !resultValue.IsNil() {
				result = resultValue.Interface().(Result)
			}
			if when == BEFORE && result != nil {
				return result
			}
		}
		return result
	}()

	if result != nil {
		c.Result = result
	}
}

var interceptors []*Interception

// Install a general interceptor.
// This can be applied to any Controller.
// It must have the signature of:
//   func example(c *rev.Controller) rev.Result
func InterceptFunc(intc InterceptorFunc, when InterceptTime, target interface{}) {
	interceptors = append(interceptors, &Interception{
		When:         when,
		function:     intc,
		callable:     reflect.ValueOf(intc),
		target:       reflect.TypeOf(target),
		interceptAll: target == ALL_CONTROLLERS,
	})
}

// Install an interceptor method that applies to its own Controller.
//   func (c AppController) example() rev.Result
//   func (c *AppController) example() rev.Result
func InterceptMethod(intc InterceptorMethod, when InterceptTime) {
	methodType := reflect.TypeOf(intc)
	if methodType.Kind() != reflect.Func || methodType.NumOut() != 1 || methodType.NumIn() != 1 {
		log.Fatalln("Interceptor method should have signature like",
			"'func (c *AppController) example() rev.Result' but was", methodType)
	}
	interceptors = append(interceptors, &Interception{
		When:     when,
		method:   intc,
		callable: reflect.ValueOf(intc),
		target:   methodType.In(0),
	})
}

func getInterceptors(when InterceptTime, val reflect.Value) []*Interception {
	result := []*Interception{}
	for _, intc := range interceptors {
		if intc.When != when {
			continue
		}

		if intc.interceptAll || findTarget(val, intc.target).IsValid() {
			result = append(result, intc)
		}
	}
	return result
}

// Find the value of the target, starting from val and including embedded types.
// Also, convert between any difference in indirection.
// If the target couldn't be found, the returned Value will have IsValid() == false
func findTarget(val reflect.Value, target reflect.Type) reflect.Value {
	// Look through the embedded types (until we reach the *rev.Controller at the top).
	for {
		// Check if val is of a similar type to the target type.
		if val.Type() == target {
			return val
		}
		if val.Kind() == reflect.Ptr && val.Elem().Type() == target {
			return val.Elem()
		}
		if target.Kind() == reflect.Ptr && target.Elem() == val.Type() {
			return val.Addr()
		}

		// If we reached the *rev.Controller and still didn't find what we were
		// looking for, give up.
		if val.Type() == controllerPtrType {
			break
		}

		// Else, drill into the first field (which had better be an embedded type).
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		val = val.Field(0)
	}

	return reflect.Value{}
}
