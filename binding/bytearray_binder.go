package binding

import (
	"reflect"
)

var (
	byteArrayBinder = Binder{bindByteArray, nil}
)


func bindByteArray(params *Params, name string, typ reflect.Type) reflect.Value {
	if reader := getMultipartFile(params, name); reader != nil {
		b, err := ioutil.ReadAll(reader)
		if err == nil {
			return reflect.ValueOf(b)
		}
		// TODO WARN.Println("Error reading uploaded file contents:", err)
	}
	return reflect.Zero(typ)
}


func init() {
	TypeBinders[reflect.TypeOf([]byte{})] = byteArrayBinder
}

