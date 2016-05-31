package binding

import (
	"reflect"
)

var (
	readerBinder = Binder{bindReader, nil}
	readSeekerBinder = Binder{bindReader, nil}
)


func bindReader(params *Params, name string, typ reflect.Type) reflect.Value {
	if reader := getMultipartFile(params, name); reader != nil {
		return reflect.ValueOf(reader.(io.ReadSeeker))
	}
	return reflect.Zero(typ)
}


func init() {
	TypeBinders[reflect.TypeOf((*io.Reader)(nil)).Elem()] = readerBinder
	TypeBinders[reflect.TypeOf((*io.ReadSeeker)(nil)).Elem()] = readSeekerBinder
}
