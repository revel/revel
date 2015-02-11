package binding

import (
	"reflect"
)

var (
	uintBinder = Binder{
		Bind: valueBinder(func(val string, typ reflect.Type) reflect.Value {
			if len(val) == 0 {
				return reflect.Zero(typ)
			}
			uintValue, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				// TODO WARN.Println(err)
				return reflect.Zero(typ)
			}
			pValue := reflect.New(typ)
			pValue.Elem().SetUint(uintValue)
			return pValue.Elem()
		}),
		Unbind: func(output map[string]string, key string, val interface{}) {
			output[key] = fmt.Sprintf("%d", val)
		},
	}
)

func init() {
	KindBinders[reflect.Uint] = uintBinder
	KindBinders[reflect.Uint8] = uintBinder
	KindBinders[reflect.Uint16] = uintBinder
	KindBinders[reflect.Uint32] = uintBinder
	KindBinders[reflect.Uint64] = uintBinder
}
