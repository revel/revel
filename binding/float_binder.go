package binding

import (
	"reflect"
)

var (
	floatBinder = Binder{
		Bind: valueBinder(func(val string, typ reflect.Type) reflect.Value {
			if len(val) == 0 {
				return reflect.Zero(typ)
			}
			floatValue, err := strconv.ParseFloat(val, 64)
			if err != nil {
				// TODO WARN.Println(err)
				return reflect.Zero(typ)
			}
			pValue := reflect.New(typ)
			pValue.Elem().SetFloat(floatValue)
			return pValue.Elem()
		}),
		Unbind: func(output map[string]string, key string, val interface{}) {
			output[key] = fmt.Sprintf("%f", val)
		},
	}
)

func init() {
	KindBinders[reflect.Float32] = floatBinder
	KindBinders[reflect.Float64] = floatBinder
}
