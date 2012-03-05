package play

import (
	"reflect"
	"strconv"
	"strings"
)

type keyValue struct {
	key, value string
}

// A Binder translates between url parameters and Go data structures.
// The caller must group together key-value pairs that correspond to a single
// argument (e.g. an array or a struct).
//
// Here is an example.
// Request: url?id=123&ol[0]=1&ol[1]=2&ul[]=str&ul[]=array&user.name=rob
// Action: Example.Action(id int, ol []int, ul []string, user User)
// Calls:
// - Binder(int,       keyValue[]{ {"id", "123"} })
// - Binder([]int,     keyValue[]{ {"ol[0]", "1"}, {"ol[1]", "2"} })
// - Binder([]string,  keyValue[]{ {"ul[]", "str"}, {"ul[]", "array"} })
// - Binder(User,      keyValue[]{ {"user.name", "rob"} })
//
// valueType is the type of value that should be returned.
// keyValues are the key/value pairs that constitute the value.
// The Go value is returned as a reflect.Value.
type Binder func(valueType reflect.Type, kv []keyValue) reflect.Value

// These are the lookups to find a Binder for any type of data.
// The most specific binder found will be used (Type before Kind)
var (
	TypeBinders = make(map[reflect.Type]Binder)
	KindBinders = make(map[reflect.Kind]Binder)
)

// Sadly, the binder lookups can not be declared initialized -- that results in
// an "initialization loop" compile error.
func init() {
	TypeBinders[reflect.TypeOf("")] = bindStr
	TypeBinders[reflect.TypeOf(0)] = bindInt

	KindBinders[reflect.Slice] = bindSlice
}

func bindStr(valueType reflect.Type, kv []keyValue) reflect.Value {
	return reflect.ValueOf(kv[0].value)
}

func bindInt(valueType reflect.Type, kv []keyValue) reflect.Value {
	intValue, err := strconv.Atoi(kv[0].value)
	if err != nil {
		LOG.Println("Error binding to int:", err)
	}
	return reflect.ValueOf(intValue)
}

// Used to keep track of the index for individual keyvalues.
type sliceValue struct {
	hasIndex bool  // true if an explicit index was assigned
	index int  // Index extracted from brackets.
	subKv []keyValue // key suffix left over, e.g. key="x[0].name" => keySuffix=".name"
}

// This function creates a slice of the given type, Binds each of the individual
// elements, and then sets them to their appropriate location in the slice.
func bindSlice(valueType reflect.Type, kvArr []keyValue) reflect.Value {
	// Map from key prefix to sub values.
	// e.g. ["foo[0].id", "foo[0].name", "bar[][0]"]
	// becomes {"foo[0]": keyValue[".id", ".name"], "bar[]": keyValue["[0]"]}
	sliceValues := make(map[string]*sliceValue)
	numNoIndex := 0
	maxIndex := -1
	for _, kv := range kvArr {
		leftBracket, rightBracket := strings.Index(kv.key, "["), strings.Index(kv.key, "]")
		if leftBracket == -1 || rightBracket == -1 {
			LOG.Println("bindSlice: missing brackets from", kv.key)
			return reflect.Zero(valueType)
		}
		index := -1
		if rightBracket > leftBracket + 1 {
			index, _ = strconv.Atoi(kv.key[leftBracket+1:rightBracket])
			if index > maxIndex {
				maxIndex = index
			}
		} else {
			numNoIndex++
		}

		// e.g. foo[0][1] breaks to prefix = "foo[0]" , suffix = "[1]"
		prefix := kv.key[:rightBracket+1]
		suffix := kv.key[rightBracket+1:]

		kv.key = suffix
		if sv, ok := sliceValues[prefix]; ok {
			sv.subKv = append(sv.subKv, kv)
			if index > sv.index {
				sv.index = index
			}
		} else {
			sliceValues[prefix] = &sliceValue{
				hasIndex: index > -1,
				index: index,
				subKv: []keyValue{kv},
			}
		}
	}

	resultArray := reflect.MakeSlice(valueType, maxIndex+1, maxIndex+1+numNoIndex)
	for _, sv := range sliceValues {
		// Recursively bind the element's value.
		elemValue := Bind(valueType.Elem(), sv.subKv)
		if sv.hasIndex {
			resultArray.Index(sv.index).Set(elemValue)
		} else {
			resultArray = reflect.Append(resultArray, elemValue)
		}
	}

	return resultArray
}

// Parse the value string into a real Go value.
// Returns 0 values when things can not be parsed.
func Bind(valueType reflect.Type, kv []keyValue) reflect.Value {
	binder, ok := TypeBinders[valueType]
	if ! ok {
		binder, ok = KindBinders[valueType.Kind()]
		if ! ok {
			LOG.Println("No binder for type:", valueType)
			return reflect.Zero(valueType)
		}
	}
	return binder(valueType, kv)
}
