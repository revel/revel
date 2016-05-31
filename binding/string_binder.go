package binding

import (
	"reflect"
)

var (
	stringBinder = Binder{
		Bind: valueBinder(func(val string, typ reflect.Type) reflect.Value {
			return reflect.ValueOf(val)
		}),
		Unbind: func(output map[string]string, name string, val interface{}) {
			output[name] = val.(string)
		},
	}
)


func init() {
	KindBinders[reflect.String] = stringBinder
}
