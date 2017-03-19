// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"reflect"
	"strings"
)

// Map from "Controller" or "Controller.Method" to the Filter chain
var filterOverrides = make(map[string][]Filter)

// FilterConfigurator allows the developer configure the filter chain on a
// per-controller or per-action basis.  The filter configuration is applied by
// the FilterConfiguringFilter, which is itself a filter stage.  For example,
//
// Assuming:
//   Filters = []Filter{
//     RouterFilter,
//     FilterConfiguringFilter,
//     SessionFilter,
//     ActionInvoker,
//   }
//
// Add:
//   FilterAction(App.Action).
//     Add(OtherFilter)
//
//   => RouterFilter, FilterConfiguringFilter, SessionFilter, OtherFilter, ActionInvoker
//
// Remove:
//   FilterAction(App.Action).
//     Remove(SessionFilter)
//
//   => RouterFilter, FilterConfiguringFilter, OtherFilter, ActionInvoker
//
// Insert:
//   FilterAction(App.Action).
//     Insert(OtherFilter, revel.BEFORE, SessionFilter)
//
//   => RouterFilter, FilterConfiguringFilter, OtherFilter, SessionFilter, ActionInvoker
//
// Filter modifications may be combined between Controller and Action.  For example:
//   FilterController(App{}).
//     Add(Filter1)
//   FilterAction(App.Action).
//     Add(Filter2)
//
//  .. would result in App.Action being filtered by both Filter1 and Filter2.
//
// Note: the last filter stage is not subject to the configurator.  In
// particular, Add() adds a filter to the second-to-last place.
type FilterConfigurator struct {
	key            string // e.g. "App", "App.Action"
	controllerName string // e.g. "App"
}

func newFilterConfigurator(controllerName, methodName string) FilterConfigurator {
	if methodName == "" {
		return FilterConfigurator{controllerName, controllerName}
	}
	return FilterConfigurator{controllerName + "." + methodName, controllerName}
}

// FilterController returns a configurator for the filters applied to all
// actions on the given controller instance.  For example:
//   FilterController(MyController{})
func FilterController(controllerInstance interface{}) FilterConfigurator {
	t := reflect.TypeOf(controllerInstance)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return newFilterConfigurator(t.Name(), "")
}

// FilterAction returns a configurator for the filters applied to the given
// controller method. For example:
//   FilterAction(MyController.MyAction)
func FilterAction(methodRef interface{}) FilterConfigurator {
	var (
		methodValue = reflect.ValueOf(methodRef)
		methodType  = methodValue.Type()
	)
	if methodType.Kind() != reflect.Func || methodType.NumIn() == 0 {
		panic("Expecting a controller method reference (e.g. Controller.Action), got a " +
			methodType.String())
	}

	controllerType := methodType.In(0)
	method := FindMethod(controllerType, methodValue)
	if method == nil {
		panic("Action not found on controller " + controllerType.Name())
	}

	for controllerType.Kind() == reflect.Ptr {
		controllerType = controllerType.Elem()
	}

	return newFilterConfigurator(controllerType.Name(), method.Name)
}

// Add the given filter in the second-to-last position in the filter chain.
// (Second-to-last so that it is before ActionInvoker)
func (conf FilterConfigurator) Add(f Filter) FilterConfigurator {
	conf.apply(func(fc []Filter) []Filter {
		return conf.addFilter(f, fc)
	})
	return conf
}

func (conf FilterConfigurator) addFilter(f Filter, fc []Filter) []Filter {
	return append(fc[:len(fc)-1], f, fc[len(fc)-1])
}

// Remove a filter from the filter chain.
func (conf FilterConfigurator) Remove(target Filter) FilterConfigurator {
	conf.apply(func(fc []Filter) []Filter {
		return conf.rmFilter(target, fc)
	})
	return conf
}

func (conf FilterConfigurator) rmFilter(target Filter, fc []Filter) []Filter {
	for i, f := range fc {
		if FilterEq(f, target) {
			return append(fc[:i], fc[i+1:]...)
		}
	}
	return fc
}

// Insert a filter into the filter chain before or after another.
// This may be called with the BEFORE or AFTER constants, for example:
//   revel.FilterAction(App.Index).
//     Insert(MyFilter, revel.BEFORE, revel.ActionInvoker).
//     Insert(MyFilter2, revel.AFTER, revel.PanicFilter)
func (conf FilterConfigurator) Insert(insert Filter, where When, target Filter) FilterConfigurator {
	if where != BEFORE && where != AFTER {
		panic("where must be BEFORE or AFTER")
	}
	conf.apply(func(fc []Filter) []Filter {
		return conf.insertFilter(insert, where, target, fc)
	})
	return conf
}

func (conf FilterConfigurator) insertFilter(insert Filter, where When, target Filter, fc []Filter) []Filter {
	for i, f := range fc {
		if FilterEq(f, target) {
			if where == BEFORE {
				return append(fc[:i], append([]Filter{insert}, fc[i:]...)...)
			}
			return append(fc[:i+1], append([]Filter{insert}, fc[i+1:]...)...)
		}
	}
	return fc
}

// getChain returns the filter chain that applies to the given controller or
// action.  If no overrides are configured, then a copy of the default filter
// chain is returned.
func (conf FilterConfigurator) getChain() []Filter {
	var filters []Filter
	if filters = getOverrideChain(conf.controllerName, conf.key); filters == nil {
		// The override starts with all filters after FilterConfiguringFilter
		for i, f := range Filters {
			if FilterEq(f, FilterConfiguringFilter) {
				filters = make([]Filter, len(Filters)-i-1)
				copy(filters, Filters[i+1:])
				break
			}
		}
		if filters == nil {
			panic("FilterConfiguringFilter not found in revel.Filters.")
		}
	}
	return filters
}

// apply applies the given functional change to the filter overrides.
// No other function modifies the filterOverrides map.
func (conf FilterConfigurator) apply(f func([]Filter) []Filter) {
	// Updates any actions that have had their filters overridden, if this is a
	// Controller configurator.
	if conf.controllerName == conf.key {
		for k, v := range filterOverrides {
			if strings.HasPrefix(k, conf.controllerName+".") {
				filterOverrides[k] = f(v)
			}
		}
	}

	// Update the Controller or Action overrides.
	filterOverrides[conf.key] = f(conf.getChain())
}

// FilterEq returns true if the two filters reference the same filter.
func FilterEq(a, b Filter) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}

// FilterConfiguringFilter is a filter stage that customizes the remaining
// filter chain for the action being invoked.
func FilterConfiguringFilter(c *Controller, fc []Filter) {
	if newChain := getOverrideChain(c.Name, c.Action); newChain != nil {
		newChain[0](c, newChain[1:])
		return
	}
	fc[0](c, fc[1:])
}

// getOverrideChain retrieves the overrides for the action that is set
func getOverrideChain(controllerName, action string) []Filter {
	if newChain, ok := filterOverrides[action]; ok {
		return newChain
	}
	if newChain, ok := filterOverrides[controllerName]; ok {
		return newChain
	}
	return nil
}
