package binding

import (
	"reflect"
)

var (
	pointerBinder = Binder{
		Bind: func(params *Params, name string, typ reflect.Type) reflect.Value {
			return Bind(params, name, typ.Elem()).Addr()
		},
		Unbind: func(output map[string]string, name string, val interface{}) {
			Unbind(output, name, reflect.ValueOf(val).Elem().Interface())
		},
	}
)

func init() {
	KindBinders[reflect.Ptr] = pointerBinder
}
