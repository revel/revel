package play

import (
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
type Interceptor func(*Controller) Result
type InterceptTime int

const (
	BEFORE InterceptTime = iota
	AFTER
)

type Interception struct {
	Func   Interceptor
	When   InterceptTime
	Target interface{}

	targetType reflect.Type
}

var interceptors []*Interception

// Install an interceptor
func Intercept(intc Interceptor, when InterceptTime, target interface{}) {
	interceptors = append(interceptors, &Interception{
		Func:       intc,
		When:       when,
		Target:     target,
		targetType: reflect.TypeOf(target),
	})
}

func getInterceptors(when InterceptTime, targetType reflect.Type) []Interceptor {
	result := []Interceptor{}
	for _, intc := range interceptors {
		if intc.When == when && intc.targetType == targetType {
			result = append(result, intc.Func)
		}
	}
	return result
}
