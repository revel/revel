package revel

import (
	"reflect"
)

// Autocalls any defined before and after methods on the target controller
// If either calls returns a value then the result is returned
func BeforeAfterFilter(c *Controller, fc []Filter) {
	defer func() {
		if resultValue := beforeAfterFilterInvoke(FINALLY, c); resultValue != nil && !resultValue.IsNil() {
			c.Result = resultValue.Interface().(Result)
		}
	}()
	defer func() {
		if err := recover(); err != nil {
			if resultValue := beforeAfterFilterInvoke(PANIC, c); resultValue != nil && !resultValue.IsNil() {
				c.Result = resultValue.Interface().(Result)
			}
			panic(err)
		}
	}()
	if resultValue := beforeAfterFilterInvoke(BEFORE, c); resultValue != nil && !resultValue.IsNil() {
		c.Result = resultValue.Interface().(Result)
	}
	fc[0](c, fc[1:])
	if resultValue := beforeAfterFilterInvoke(AFTER, c); resultValue != nil && !resultValue.IsNil() {
		c.Result = resultValue.Interface().(Result)
	}
}

func beforeAfterFilterInvoke(method When, c *Controller) (r *reflect.Value) {

	if c.Type == nil {
		return
	}
	var index []*ControllerFieldPath
	switch method {
	case BEFORE:
		index = c.Type.ControllerEvents.Before
	case AFTER:
		index = c.Type.ControllerEvents.After
	case FINALLY:
		index = c.Type.ControllerEvents.Finally
	case PANIC:
		index = c.Type.ControllerEvents.Panic
	}

	if len(index) == 0 {
		return
	}
	for _, function := range index {
		result := function.Invoke(reflect.ValueOf(c.AppController), nil)[0]
		if !result.IsNil() {
			return &result
		}
	}

	return
}
