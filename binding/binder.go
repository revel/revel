package binding

import (
	"reflect"
	"strings"
)

// A Binder translates between string parameters from a URL and Go data structures.
type Binder struct {
	// Bind takes the name and type of the desired parameter and constructs it
	// from one or more values from Params.
	//
	// Example
	//
	// Request:
	//   url?id=123&ol[0]=1&ol[1]=2&ul[]=str&ul[]=array&user.Name=rob
	//
	// Action:
	//   Example.Action(id int, ol []int, ul []string, user User)
	//
	// Calls:
	//   Bind(params, "id", int): 123
	//   Bind(params, "ol", []int): {1, 2}
	//   Bind(params, "ul", []string): {"str", "array"}
	//   Bind(params, "user", User): User{Name:"rob"}
	//
	// Note that only exported struct fields may be bound.
	Bind func(params *Params, name string, typ reflect.Type) reflect.Value

	// Unbind serializes a given value to one or more URL parameters of the given
	// name.
	Unbind func(output map[string]string, name string, val interface{})
}


var (
	// These are the lookups to find a Binder for any type of data.
	// The most specific binder found will be used (Type before Kind)
	TypeBinders = make(map[reflect.Type]Binder)
	KindBinders = make(map[reflect.Kind]Binder)

	// Applications can add custom time formats to this array, and they will be
	// automatically attempted when binding a time.Time.
	TimeFormats = []string{}
	DateFormat     string
	DateTimeFormat string
)


// An adapter for easily making one-key-value binders.
func valueBinder(f func(value string, typ reflect.Type) reflect.Value) func(*Params, string, reflect.Type) reflect.Value {
	return func(params *Params, name string, typ reflect.Type) reflect.Value {
		vals, ok := params.Values[name]
		if !ok || len(vals) == 0 {
			return reflect.Zero(typ)
		}
		return f(vals[0], typ)
	}
}


// Break on dots and brackets.
// e.g. bar => "bar", bar.baz => "bar", bar[0] => "bar"
func nextKey(key string) string {
	fieldLen := strings.IndexAny(key, ".[")
	if fieldLen == -1 {
		return key
	}
	return key[:fieldLen]
}
