package binding

import (
	"reflect"
)

var (
	timeBinder = Binder{
		Bind: valueBinder(func(val string, typ reflect.Type) reflect.Value {
			for _, f := range TimeFormats {
				if r, err := time.Parse(f, val); err == nil {
					return reflect.ValueOf(r)
				}
			}
			return reflect.Zero(typ)
		}),
		Unbind: func(output map[string]string, name string, val interface{}) {
			var (
				t       = val.(time.Time)
				format  = DateTimeFormat
				h, m, s = t.Clock()
			)
			if h == 0 && m == 0 && s == 0 {
				format = DateFormat
			}
			output[name] = t.Format(format)
		},
	}
)

func init() {
	TypeBinders[reflect.TypeOf(time.Time{})] = timeBinder
}
