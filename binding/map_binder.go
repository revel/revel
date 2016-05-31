package binding

import (
	"reflect"
)

var (
	mapBinder = Binder{
		Bind:   bindMap,
		Unbind: unbindMap,
	}
)


// bindMap converts parameters using map syntax into the corresponding map. e.g.:
//   params["a[5]"]=foo, name="a", typ=map[int]string => map[int]string{5: "foo"}
func bindMap(params *Params, name string, typ reflect.Type) reflect.Value {
	var (
		result    = reflect.MakeMap(typ)
		keyType   = typ.Key()
		valueType = typ.Elem()
	)
	for paramName, values := range params.Values {
		if !strings.HasPrefix(paramName, name+"[") || paramName[len(paramName)-1] != ']' {
			continue
		}

		key := paramName[len(name)+1 : len(paramName)-1]
		result.SetMapIndex(BindValue(key, keyType), BindValue(values[0], valueType))
	}
	return result
}





func unbindMap(output map[string]string, name string, iface interface{}) {
	mapValue := reflect.ValueOf(iface)
	for _, key := range mapValue.MapKeys() {
		Unbind(output, name+"["+fmt.Sprintf("%v", key.Interface())+"]",
			mapValue.MapIndex(key).Interface())
	}
}

func init() {
	KindBinders[reflect.Map] = mapBinder
}
