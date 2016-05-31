package binding

import (
	"reflect"
)

var (
	// Booleans support a couple different value formats:
	// "true" and "false"
	// "on" and "" (a checkbox)
	// "1" and "0" (why not)
	BoolBinder = Binder{
		Bind: valueBinder(func(val string, typ reflect.Type) reflect.Value {
			v := strings.TrimSpace(strings.ToLower(val))
			switch v {
			case "true", "on", "1":
				return reflect.ValueOf(true)
			}
			// Return false by default.
			return reflect.ValueOf(false)
		}),
		Unbind: func(output map[string]string, name string, val interface{}) {
			output[name] = fmt.Sprintf("%t", val)
		},
	}
)

func init() {
	KindBinders[reflect.Bool] = boolBinder
}
