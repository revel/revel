package revel

import "reflect"

// Map from "Controller" or "Controller.Method" to FilterChain
var filterOverrides = make(map[string]FilterChain)

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
// Set:
//   FilterAction(App.Action).
//     Set(SessionFilter, OtherFilter)
//
//   => RouterFilter, FilterConfiguringFilter, SessionFilter, OtherFilter, ActionInvoker
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
	} else {
		return FilterConfigurator{controllerName + "." + methodName, controllerName}
	}
}

// FilterController returns a configurator for the filters applied to all
// actions on the given controller instance.  For example:
//   FilterAction(MyController{})
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
	fc := conf.getOverrideFilters()
	filterOverrides[conf.key] = append(fc[:len(fc)-1], f, fc[len(fc)-1])
	return conf
}

// Remove a filter from the filter chain.
func (conf FilterConfigurator) Remove(target Filter) FilterConfigurator {
	var (
		targetType = reflect.TypeOf(target)
		filters    = conf.getOverrideFilters()
	)
	for i, f := range filters {
		if reflect.TypeOf(f) == targetType {
			filterOverrides[conf.key] = append(filters[:i], filters[i+1:]...)
			return conf
		}
	}
	panic("Did not find target filter: " + targetType.Name())
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
	var (
		targetType = reflect.TypeOf(target)
		filters    = conf.getOverrideFilters()
	)
	for i, f := range filters {
		if reflect.TypeOf(f) == targetType {
			filterOverrides[conf.key] = append(filters[:i], append([]Filter{insert}, filters[i:]...)...)
			return conf
		}
	}
	panic("Did not find target filter: " + targetType.Name())
}

// getOverrideFilters returns the filter chain that applies to the given
// controller or action.  If no overrides are configured, then a copy of the
// default filter chain is returned.
func (conf FilterConfigurator) getOverrideFilters() []Filter {
	var (
		filters []Filter
		ok      bool
	)
	filters, ok = filterOverrides[conf.key]
	if !ok {
		filters, ok = filterOverrides[conf.controllerName]
		if !ok {
			// The override starts with all filters after FilterConfiguringFilter
			for i, f := range Filters {
				if f == FilterConfiguringFilter {
					filters = make([]Filter, len(Filters)-i-1)
					copy(filters, Filters[i+1:])
					break
				}
			}
			if filters == nil {
				panic("FilterConfiguringFilter not found in revel.Filters.")
			}
		}
	}
	return filters
}

// FilterConfiguringFilter is a filter stage that customizes the remaining
// filter chain for the action being invoked.
var FilterConfiguringFilter filterConfiguringFilter

type filterConfiguringFilter struct{}

func (f filterConfiguringFilter) Call(c *Controller, fc FilterChain) {
	if newChain, ok := filterOverrides[c.Name+"."+c.Action]; ok {
		newChain[0].Call(c, newChain[1:])
		return
	}

	if newChain, ok := filterOverrides[c.Name]; ok {
		newChain[0].Call(c, newChain[1:])
		return
	}

	fc[0].Call(c, fc[1:])
}
