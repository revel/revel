// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"log"
	"reflect"
)

// An InterceptorFunc is functionality invoked by the framework BEFORE or AFTER
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
//   func example(*revel.Controller) revel.Result
//
// Method Interceptors are provided so that properties can be set on application
// controllers.
//
//   func (c AppController) example() revel.Result
//   func (c *AppController) example() revel.Result
//
type InterceptorFunc func(*Controller) Result
type InterceptorMethod interface{}
type When int

const (
	BEFORE When = iota
	AFTER
	PANIC
	FINALLY
)

type InterceptTarget int

const (
	AllControllers InterceptTarget = iota
)

type Interception struct {
	When When

	function InterceptorFunc
	method   InterceptorMethod

	callable     reflect.Value
	target       reflect.Type
	interceptAll bool
}

type interceptionEntry struct {
	intc   *Interception
	target reflect.Value
}

// Invoke performs the given interception.
// val is a pointer to the App Controller.
func (i Interception) Invoke(val, target reflect.Value) reflect.Value {
	if i.function != nil {
		// If it's an InterceptorFunc, then the type must be *Controller.
		// We can find that by following the embedded types up the chain.
		for val.Type() != controllerPtrType {
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			val = val.Field(0)
		}
		target = val
	}

	vals := i.callable.Call([]reflect.Value{target})
	return vals[0]
}

func InterceptorFilter(c *Controller, fc []Filter) {
	defer invokeInterceptors(FINALLY, c)
	defer func() {
		if err := recover(); err != nil {
			invokeInterceptors(PANIC, c)
			panic(err)
		}
	}()

	// Invoke the BEFORE interceptors and return early, if we get a result.
	invokeInterceptors(BEFORE, c)
	if c.Result != nil {
		return
	}

	fc[0](c, fc[1:])
	invokeInterceptors(AFTER, c)
}

func invokeInterceptors(when When, c *Controller) {
	var (
		app    = reflect.ValueOf(c.AppController)
		result Result
	)
	for _, interceptionEntry := range getInterceptorEntries(when, app) {
		resultValue := interceptionEntry.intc.Invoke(app, interceptionEntry.target)
		if !resultValue.IsNil() {
			result = resultValue.Interface().(Result)
		}
		if when == BEFORE && result != nil {
			c.Result = result
			return
		}
	}
	if result != nil {
		c.Result = result
	}
}

var interceptors []*Interception

// InterceptFunc installs a general interceptor.
// This can be applied to any Controller.
// It must have the signature of:
//   func example(c *revel.Controller) revel.Result
func InterceptFunc(intc InterceptorFunc, when When, target interface{}) {
	interceptors = append(interceptors, &Interception{
		When:         when,
		function:     intc,
		callable:     reflect.ValueOf(intc),
		target:       reflect.TypeOf(target),
		interceptAll: target == AllControllers,
	})
}

// InterceptMethod installs an interceptor method that applies to its own Controller.
//   func (c AppController) example() revel.Result
//   func (c *AppController) example() revel.Result
func InterceptMethod(intc InterceptorMethod, when When) {
	methodType := reflect.TypeOf(intc)
	if methodType.Kind() != reflect.Func || methodType.NumOut() != 1 || methodType.NumIn() != 1 {
		log.Fatalln("Interceptor method should have signature like",
			"'func (c *AppController) example() revel.Result' but was", methodType)
	}
	interceptors = append(interceptors, &Interception{
		When:     when,
		method:   intc,
		callable: reflect.ValueOf(intc),
		target:   methodType.In(0),
	})
}

func getInterceptorEntries(when When, val reflect.Value) []*interceptionEntry {
	result := []*interceptionEntry{}
	for _, intc := range interceptors {
		if intc.When != when {
			continue
		}

		v := findTarget(val, intc.target)
		if intc.interceptAll || v.IsValid() {
			result = append(result, &interceptionEntry{intc, v})
		}
	}
	return result
}

// Find the value of the target, starting from val and including embedded types.
// Also, convert between any difference in indirection.
// If the target couldn't be found, the returned Value will have IsValid() == false
func findTarget(val reflect.Value, target reflect.Type) reflect.Value {
	// Look through the embedded types (until we reach the *revel.Controller at the top).
	valueQueue := []reflect.Value{val}
	for len(valueQueue) > 0 {
		val, valueQueue = valueQueue[0], valueQueue[1:]

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

		// If we reached the *revel.Controller and still didn't find what we were
		// looking for, give up.
		if val.Type() == controllerPtrType {
			continue
		}

		// Else, add each anonymous field to the queue.
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		for i := 0; i < val.NumField(); i++ {
			if val.Type().Field(i).Anonymous {
				valueQueue = append(valueQueue, val.Field(i))
			}
		}
	}

	return reflect.Value{}
}
