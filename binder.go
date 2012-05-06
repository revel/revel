package rev

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

// A Binder translates between string parameters and Go data structures.
//
// Here is an example.
//
// Request:
//   url?id=123&ol[0]=1&ol[1]=2&ul[]=str&ul[]=array&user.Name=rob
// Action:
//   Example.Action(id int, ol []int, ul []string, user User)
// Calls:
//   Binder(params, "id", int): 123
//   Binder(params, "ol", []int): {1, 2}
//   Binder(params, "ul", []string): {"str", "array"}
//   Binder(params, "user", User): User{Name:"rob"}
//
// Note that only exported struct fields may be bound.
type Binder func(params map[string][]string, name string, typ reflect.Type) reflect.Value

// An adapter for easily making one-key-value binders.
func SimpleBinder(f func(value string, typ reflect.Type) reflect.Value) Binder {
	return func(params map[string][]string, name string, typ reflect.Type) reflect.Value {
		vals, ok := params[name]
		if !ok || len(vals) == 0 {
			return reflect.Zero(typ)
		}
		return f(vals[0], typ)
	}
}

// These are the lookups to find a Binder for any type of data.
// The most specific binder found will be used (Type before Kind)
var (
	TypeBinders = make(map[reflect.Type]Binder)
	KindBinders = make(map[reflect.Kind]Binder)
)

// Sadly, the binder lookups can not be declared initialized -- that results in
// an "initialization loop" compile error.
func init() {
	intBinder := SimpleBinder(bindInt)

	KindBinders[reflect.Int] = intBinder
	KindBinders[reflect.Int8] = intBinder
	KindBinders[reflect.Int16] = intBinder
	KindBinders[reflect.Int32] = intBinder
	KindBinders[reflect.Int64] = intBinder

	KindBinders[reflect.String] = SimpleBinder(bindStr)
	KindBinders[reflect.Bool] = SimpleBinder(bindBool)
	KindBinders[reflect.Slice] = bindSlice
	KindBinders[reflect.Struct] = bindStruct
	KindBinders[reflect.Ptr] = bindPointer

	TypeBinders[reflect.TypeOf(time.Time{})] = SimpleBinder(bindTime)
}

var (
	// Applications can add custom time formats to this array, and they will be
	// automatically attempted when binding a time.Time.
	TimeFormats = []string{"2006-01-02", "2006-01-02 15:04"}
)

func bindStr(val string, typ reflect.Type) reflect.Value {
	return reflect.ValueOf(val)
}

func bindInt(val string, typ reflect.Type) reflect.Value {
	intValue, err := strconv.Atoi(val)
	if err != nil {
		LOG.Println("BindInt:", err)
	}
	return reflect.ValueOf(intValue)
}

// Booleans support a couple different value formats:
// "true" and "false"
// "on" and "" (a checkbox)
// "1" and "0" (why not)
func bindBool(val string, typ reflect.Type) reflect.Value {
	v := strings.TrimSpace(strings.ToLower(val))
	switch v {
	case "true", "on", "1":
		return reflect.ValueOf(true)
	}
	// Return false by default.
	return reflect.ValueOf(false)
}

// Used to keep track of the index for individual keyvalues.
type sliceValue struct {
	index int           // Index extracted from brackets.  If -1, no index was provided.
	value reflect.Value // the bound value for this slice element.
}

// This function creates a slice of the given type, Binds each of the individual
// elements, and then sets them to their appropriate location in the slice.
// If elements are provided without an explicit index, they are added (in
// unspecified order) to the end of the slice.
func bindSlice(params map[string][]string, name string, typ reflect.Type) reflect.Value {
	// Collect an array of slice elements with their indexes (and the max index).
	maxIndex := -1
	numNoIndex := 0
	sliceValues := []sliceValue{}
	for key, vals := range params {
		for _, val := range vals {
			if !strings.HasPrefix(key, name+"[") {
				continue
			}

			// Extract the index and add it to the sliceValues array.
			leftBracket, rightBracket := len(name), strings.Index(key[len(name):], "]")+len(name)
			if rightBracket == -1 {
				continue
			}

			index := -1
			if rightBracket > leftBracket+1 {
				index, _ = strconv.Atoi(key[leftBracket+1 : rightBracket])
				if index > maxIndex {
					maxIndex = index
				}
				sliceValues = append(sliceValues, sliceValue{
					index: index,
					value: Bind(params, key[:rightBracket+1], typ.Elem()),
				})
			} else {
				// Unindexed elements can only be direct-bound.
				numNoIndex++
				sliceValues = append(sliceValues, sliceValue{
					index: -1,
					value: SimpleBind(val, typ.Elem()),
				})
			}
		}
	}

	resultArray := reflect.MakeSlice(typ, maxIndex+1, maxIndex+1+numNoIndex)
	for _, sv := range sliceValues {
		if sv.index != -1 {
			resultArray.Index(sv.index).Set(sv.value)
		} else {
			resultArray = reflect.Append(resultArray, sv.value)
		}
	}

	return resultArray
}

func bindStruct(params map[string][]string, name string, typ reflect.Type) reflect.Value {
	result := reflect.New(typ).Elem()
	fieldValues := make(map[string]reflect.Value)
	for key, _ := range params {
		if !strings.HasPrefix(key, name+".") {
			continue
		}

		// Get the name of the struct property.
		// Strip off the prefix. e.g. foo.bar.baz => bar.baz
		suffix := key[len(name)+1:]

		// Break it on dots and brackets.
		// e.g. bar => "bar", bar.baz => "bar", bar[0] => "bar"
		fieldLen := strings.IndexAny(suffix, ".[")
		if fieldLen == -1 {
			fieldLen = len(suffix)
		}
		fieldName := suffix[:fieldLen]

		if _, ok := fieldValues[fieldName]; !ok {
			// Time to bind this field.  Get it and make sure we can set it.
			fieldValue := result.FieldByName(fieldName)
			if !fieldValue.IsValid() {
				LOG.Println("W: bindStruct: Field not found:", fieldName)
				continue
			}
			if !fieldValue.CanSet() {
				LOG.Println("W: bindStruct: Field not settable:", fieldName)
				continue
			}
			boundVal := Bind(params, key[:len(name)+1+fieldLen], fieldValue.Type())
			fieldValue.Set(boundVal)
			fieldValues[fieldName] = boundVal
		}
	}

	return result
}

func bindPointer(params map[string][]string, name string, typ reflect.Type) reflect.Value {
	return Bind(params, name, typ.Elem()).Addr()
}

// This expects a single keyValue.
func bindTime(val string, typ reflect.Type) reflect.Value {
	for _, f := range TimeFormats {
		if r, err := time.Parse(f, val); err == nil {
			return reflect.ValueOf(r)
		}
	}
	return reflect.Zero(typ)
}

// Parse the value string into a real Go value.
// Returns 0 values when things can not be parsed.
func Bind(params map[string][]string, name string, typ reflect.Type) reflect.Value {
	// If there's no data, just return the type's zero.
	if len(params) == 0 {
		return reflect.Zero(typ)
	}

	binder, ok := TypeBinders[typ]
	if !ok {
		binder, ok = KindBinders[typ.Kind()]
		if !ok {
			LOG.Println("No binder for type:", typ)
			return reflect.Zero(typ)
		}
	}
	return binder(params, name, typ)
}

func SimpleBind(val string, typ reflect.Type) reflect.Value {
	return Bind(map[string][]string{"": {val}}, "", typ)
}
