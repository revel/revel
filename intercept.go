// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"log"
	"reflect"
	"sort"
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

// Invoke performs the given interception.
// val is a pointer to the App Controller.
func (i Interception) Invoke(val reflect.Value, target *reflect.Value) reflect.Value {
	var arg reflect.Value
	if i.function == nil {
		// If it's an InterceptorMethod, then we have to pass in the target type.
		arg = *target
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

	for _, intc := range getInterceptors(when, app) {
		resultValue := intc.Interceptor.Invoke(app, &intc.Target)
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

// This item is used to provide a sortable set to be returned to the caller. This ensures calls order is maintained
//
type interceptorItem struct {
	Interceptor *Interception
	Target      reflect.Value
	Level       int
}
type interceptorItemList []*interceptorItem

func (a interceptorItemList) Len() int           { return len(a) }
func (a interceptorItemList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a interceptorItemList) Less(i, j int) bool { return a[i].Level < a[j].Level }

type reverseInterceptorItemList []*interceptorItem

func (a reverseInterceptorItemList) Len() int           { return len(a) }
func (a reverseInterceptorItemList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a reverseInterceptorItemList) Less(i, j int) bool { return a[i].Level > a[j].Level }
func getInterceptors(when When, val reflect.Value) interceptorItemList {
	result := interceptorItemList{}
	for _, intc := range interceptors {
		if intc.When != when {
			continue
		}

		level, target := findTarget(val, intc.target)
		if intc.interceptAll || target.IsValid() {
			result = append(result, &interceptorItem{intc, target, level})
		}
	}

	// Before is deepest to highest
	if when == BEFORE {
		sort.Sort(result)
	} else {
		// Everything else is highest to deepest
		sort.Sort(reverseInterceptorItemList(result))
	}
	return result
}

// Find the value of the target, starting from val and including embedded types.
// Also, convert between any difference in indirection.
// If the target couldn't be found, the returned Value will have IsValid() == false
func findTarget(val reflect.Value, target reflect.Type) (int, reflect.Value) {
	// Look through the embedded types (until we reach the *revel.Controller at the top).
	valueQueue := []reflect.Value{val}
	level := 0
	for len(valueQueue) > 0 {
		val, valueQueue = valueQueue[0], valueQueue[1:]

		// Check if val is of a similar type to the target type.
		if val.Type() == target {
			return level, val
		}
		if val.Kind() == reflect.Ptr && val.Elem().Type() == target {
			return level, val.Elem()
		}
		if target.Kind() == reflect.Ptr && target.Elem() == val.Type() {
			return level, val.Addr()
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
		level--
	}

	return level, reflect.Value{}
}
