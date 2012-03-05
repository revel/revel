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
// - Binder(User,      keyValue[]{ {"user.Name", "rob"} })
//
// Note that only exported struct fields may be bound.
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
	KindBinders[reflect.Int] = bindInt
	KindBinders[reflect.Int8] = bindInt
	KindBinders[reflect.Int16] = bindInt
	KindBinders[reflect.Int32] = bindInt
	KindBinders[reflect.Int64] = bindInt

	KindBinders[reflect.String] = bindStr
	KindBinders[reflect.Bool] = bindBool
	KindBinders[reflect.Slice] = bindSlice
	KindBinders[reflect.Struct] = bindStruct
	KindBinders[reflect.Ptr] = bindPointer
}

func bindStr(valueType reflect.Type, kv []keyValue) reflect.Value {
	return reflect.ValueOf(kv[0].value)
}

func bindInt(valueType reflect.Type, kv []keyValue) reflect.Value {
	intValue, err := strconv.Atoi(kv[0].value)
	if err != nil {
		LOG.Println("Error binding", kv[0].key, ":", err)
	}
	return reflect.ValueOf(intValue)
}

// Booleans support a couple different value formats:
// "true" and "false"
// "on" and "" (a checkbox)
// "1" and "0" (why not)
func bindBool(valueType reflect.Type, kv []keyValue) reflect.Value {
	v := strings.TrimSpace(
		strings.ToLower(kv[0].value))
	switch v {
	case "true", "on", "1":
		return reflect.ValueOf(true)
	}
	// Return false by default.
	return reflect.ValueOf(false)
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

func bindStruct(valueType reflect.Type, kvArr []keyValue) reflect.Value {
	// Map from field name (e.g. key="x.name.first" => "name") to key values for that field.
	structValues := make(map[string][]keyValue)
	for _, kv := range kvArr {
		// Ignore everything up to the first dot.
		// e.g. foo.bar.baz => bar.baz
		dot := strings.Index(kv.key, ".")
		if dot == -1 {
			LOG.Println("bindStruct: missing dot", kv.key)
			return reflect.Zero(valueType)
		}
		subKey := kv.key[dot+1:]

		// Break subKey into prefix and suffix, on dots and brackets.
		// e.g. bar.baz breaks to prefix = "bar" , suffix = ".baz"
		// e.g. bar[0] breaks to prefix = "bar" , suffix = "[0]"
		prefixLen := strings.IndexAny(subKey, ".[")
		if prefixLen == -1 {
			prefixLen = len(subKey)
		}
		prefix := subKey[:prefixLen]
		suffix := subKey[prefixLen:]

		// TODO: This part of grouping args will be shared to any callers of Bind.
		kv.key = suffix
		if sv, ok := structValues[prefix]; ok {
			structValues[prefix] = append(sv, kv)
		} else {
			structValues[prefix] = []keyValue{kv}
		}
	}

	result := reflect.New(valueType).Elem()
	for fieldName, subKv := range structValues {
		// Find the field to bind.
		fieldValue := result.FieldByName(fieldName)
		if ! fieldValue.IsValid() {
			LOG.Println("bindStruct: Field not found:", fieldName)
			continue
		}
		if ! fieldValue.CanSet() {
			LOG.Println("bindStruct: Field not settable:", fieldName)
			continue
		}

		// Bind it
		fieldValue.Set(Bind(fieldValue.Type(), subKv))
	}
	return result
}

func bindPointer(valueType reflect.Type, kvArr []keyValue) reflect.Value {
	return Bind(valueType.Elem(), kvArr).Addr()
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
