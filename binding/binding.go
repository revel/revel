// binding is responsible for taking HTTP parmters and binding them to variables for specific Go types
package binding

import (
	"mime/multipart"
	"reflect"
)


// Bind takes the name and type of the desired parameter and constructs it
// from one or more values from Params.
// Returns the zero value of the type upon any sort of failure.
func Bind(params *map[string][]string, name string, typ reflect.Type) reflect.Value {
	if binder, found := binderForType(typ); found {
		return binder.Bind(params, name, typ)
	}
	return reflect.Zero(typ)
}


func BindValue(val string, typ reflect.Type) reflect.Value {
	return Bind(map[string][]string{"": {val}}, "", typ)
}


func BindFile(fileHeader *multipart.FileHeader, typ reflect.Type) reflect.Value {
	return Bind(&Params{Files: map[string][]*multipart.FileHeader{"": {fileHeader}}}, "", typ)
}


func Unbind(output map[string]string, name string, val interface{}) {
	if binder, found := binderForType(reflect.TypeOf(val)); found {
		if binder.Unbind != nil {
			binder.Unbind(output, name, val)
		} else {
			// TODO ERROR.Printf("revel/binder: can not unbind %s=%s", name, val)
		}
	}
}

// Purge frees up any temporary resources used by binder drivers during data binding
func Purge() (err error) {
	for _, binder := range TypeBinders {
		binder.Purge()
	}
	for _, binder := range KindBinders {
		binder.Purge()
	}
}


func binderForType(typ reflect.Type) (Binder, bool) {
	binder, ok := TypeBinders[typ]
	if !ok {
		binder, ok = KindBinders[typ.Kind()]
		if !ok {
			// TODO WARN.Println("revel/binder: no binder for type:", typ)
			return Binder{}, false
		}
	}
	return binder, true
}
