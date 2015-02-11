package binding

import (
	"reflect"
)

var (
	intBinder = Binder{
		Bind: valueBinder(func(val string, typ reflect.Type) reflect.Value {
			if len(val) == 0 {
				return reflect.Zero(typ)
			}
			intValue, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				// TODO WARN.Println(err)
				return reflect.Zero(typ)
			}
			pValue := reflect.New(typ)
			pValue.Elem().SetInt(intValue)
			return pValue.Elem()
		}),
		Unbind: func(output map[string]string, key string, val interface{}) {
			output[key] = fmt.Sprintf("%d", val)
		},
	}
)

func init() {
	KindBinders[reflect.Int] = intBinder
	KindBinders[reflect.Int8] = intBinder
	KindBinders[reflect.Int16] = intBinder
	KindBinders[reflect.Int32] = intBinder
	KindBinders[reflect.Int64] = intBinder
}
