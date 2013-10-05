package revel

import (
	"reflect"
)

type ActionRestriction struct {
	ControllerName string
	ActionName     string
	Check          interface{} // func (SomeControllerOrControllerPtr) Result
}

var ActionRestrictions []ActionRestriction = []ActionRestriction{}

// Calls function f (func(SomeControllerType) Result) with a controller of needed type,
// doing breadth-first search for parent controllers.
func callWithController(f interface{}, ctrlInstance interface{}) (resultValue reflect.Value) {
	appController := reflect.ValueOf(ctrlInstance).Elem()
	function := reflect.ValueOf(f)
	functionType := function.Type()

	// f must be a function
	if functionType.Kind() != reflect.Func {
		WARN.Printf("ActionRestriction check function is not a function")
		return
	}

	functionParamType := functionType.In(0)

	// f signature must be func (SomeControllerOrControllerPtr) Result
	if functionType.NumIn() != 1 {
		WARN.Printf("ActionRestriction check function must accept but one parameter")
		return
	}
	if functionType.NumOut() != 1 || functionType.Out(0) != reflect.TypeOf((*Result)(nil)).Elem() {
		WARN.Printf("ActionRestriction check function must return revel.Result")
		return
	}

	// Search for a needed controller.
	queue := []reflect.Value{appController}
	for len(queue) > 0 {
		// Get the next value and de-reference it if necessary.
		var (
			elem     = queue[0]
			elemType = elem.Type()
		)
		if elemType.Kind() == reflect.Ptr {
			elem = elem.Elem()
			elemType = elem.Type()
		}

		// If types do match, call the function and return its result.
		if elemType == functionParamType {
			return function.Call([]reflect.Value{elem})[0]
		} else if reflect.PtrTo(elemType) == functionParamType {
			return function.Call([]reflect.Value{elem.Addr()})[0]
		}

		queue = queue[1:]

		// Look at all the struct fields.
		for i := 0; i < elem.NumField(); i++ {
			// If this is not an anonymous field, skip it.
			structField := elemType.Field(i)
			if !structField.Anonymous {
				continue
			}

			queue = append(queue, elem.Field(i))
		}

	}
	WARN.Printf("Controller of type %s was not found", functionParamType)
	return
}

func AuthFilter(c *Controller, fc []Filter) {
	if c.Name != "Static" {
		for i := 0; i < len(ActionRestrictions); i++ {
			rst := ActionRestrictions[i]
			if c.Name == rst.ControllerName && (c.MethodName == rst.ActionName || rst.ActionName == "*") {
				resultValue := callWithController(rst.Check, c.AppController)
				if resultValue.IsValid() && !resultValue.IsNil() {
					c.Result = resultValue.Interface().(Result)
					return
				}
			}
		}
	}

	fc[0](c, fc[1:]) // Execute the next filter stage.
}
