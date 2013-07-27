package revel

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// A Binder translates between string parameters and Go data structures.
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

// An adapter for easily making one-key-value binders.
func ValueBinder(f func(value string, typ reflect.Type) reflect.Value) func(*Params, string, reflect.Type) reflect.Value {
	return func(params *Params, name string, typ reflect.Type) reflect.Value {
		vals, ok := params.Values[name]
		if !ok || len(vals) == 0 {
			return reflect.Zero(typ)
		}
		return f(vals[0], typ)
	}
}

const (
	DEFAULT_DATE_FORMAT     = "2006-01-02"
	DEFAULT_DATETIME_FORMAT = "2006-01-02 15:04"
)

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

	IntBinder = Binder{
		Bind: ValueBinder(func(val string, typ reflect.Type) reflect.Value {
			if len(val) == 0 {
				return reflect.Zero(typ)
			}
			intValue, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				WARN.Println(err)
				return reflect.Zero(typ)
			}
			pValue := reflect.New(typ)
			pValue.Elem().SetInt(intValue)
			return pValue.Elem()
		}),
		Unbind: func(output map[string]string, key string, val interface{}) {
			output[key] = fmt.Sprintf("%d", val)
		},
	}

	UintBinder = Binder{
		Bind: ValueBinder(func(val string, typ reflect.Type) reflect.Value {
			if len(val) == 0 {
				return reflect.Zero(typ)
			}
			uintValue, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				WARN.Println(err)
				return reflect.Zero(typ)
			}
			pValue := reflect.New(typ)
			pValue.Elem().SetUint(uintValue)
			return pValue.Elem()
		}),
		Unbind: func(output map[string]string, key string, val interface{}) {
			output[key] = fmt.Sprintf("%d", val)
		},
	}

	FloatBinder = Binder{
		Bind: ValueBinder(func(val string, typ reflect.Type) reflect.Value {
			if len(val) == 0 {
				return reflect.Zero(typ)
			}
			floatValue, err := strconv.ParseFloat(val, 64)
			if err != nil {
				WARN.Println(err)
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

	StringBinder = Binder{
		Bind: ValueBinder(func(val string, typ reflect.Type) reflect.Value {
			return reflect.ValueOf(val)
		}),
		Unbind: func(output map[string]string, name string, val interface{}) {
			output[name] = val.(string)
		},
	}

	// Booleans support a couple different value formats:
	// "true" and "false"
	// "on" and "" (a checkbox)
	// "1" and "0" (why not)
	BoolBinder = Binder{
		Bind: ValueBinder(func(val string, typ reflect.Type) reflect.Value {
			v := strings.TrimSpace(strings.ToLower(val))
			switch v {
			case "true", "on", "1":
				return reflect.ValueOf(true)
			}
			// Return false by default.
			return reflect.ValueOf(false)
		}),
		Unbind: func(output map[string]string, name string, val interface{}) {
			output[name] = fmt.Sprintf("%t", val)
		},
	}

	PointerBinder = Binder{
		Bind: func(params *Params, name string, typ reflect.Type) reflect.Value {
			return Bind(params, name, typ.Elem()).Addr()
		},
		Unbind: func(output map[string]string, name string, val interface{}) {
			Unbind(output, name, reflect.ValueOf(val).Elem().Interface())
		},
	}

	TimeBinder = Binder{
		Bind: ValueBinder(func(val string, typ reflect.Type) reflect.Value {
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

	MapBinder = Binder{
		Bind:   bindMap,
		Unbind: unbindMap,
	}
)

// Sadly, the binder lookups can not be declared initialized -- that results in
// an "initialization loop" compile error.
func init() {
	KindBinders[reflect.Int] = IntBinder
	KindBinders[reflect.Int8] = IntBinder
	KindBinders[reflect.Int16] = IntBinder
	KindBinders[reflect.Int32] = IntBinder
	KindBinders[reflect.Int64] = IntBinder

	KindBinders[reflect.Uint] = UintBinder
	KindBinders[reflect.Uint8] = UintBinder
	KindBinders[reflect.Uint16] = UintBinder
	KindBinders[reflect.Uint32] = UintBinder
	KindBinders[reflect.Uint64] = UintBinder

	KindBinders[reflect.Float32] = FloatBinder
	KindBinders[reflect.Float64] = FloatBinder

	KindBinders[reflect.String] = StringBinder
	KindBinders[reflect.Bool] = BoolBinder
	KindBinders[reflect.Slice] = Binder{bindSlice, unbindSlice}
	KindBinders[reflect.Struct] = Binder{bindStruct, unbindStruct}
	KindBinders[reflect.Ptr] = PointerBinder
	KindBinders[reflect.Map] = MapBinder

	TypeBinders[reflect.TypeOf(time.Time{})] = TimeBinder

	// Uploads
	TypeBinders[reflect.TypeOf(&os.File{})] = Binder{bindFile, nil}
	TypeBinders[reflect.TypeOf([]byte{})] = Binder{bindByteArray, nil}
	TypeBinders[reflect.TypeOf((*io.Reader)(nil)).Elem()] = Binder{bindReadSeeker, nil}
	TypeBinders[reflect.TypeOf((*io.ReadSeeker)(nil)).Elem()] = Binder{bindReadSeeker, nil}

	OnAppStart(func() {
		DateTimeFormat = Config.StringDefault("format.datetime", DEFAULT_DATETIME_FORMAT)
		DateFormat = Config.StringDefault("format.date", DEFAULT_DATE_FORMAT)
		TimeFormats = append(TimeFormats, DateTimeFormat, DateFormat)
	})
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
func bindSlice(params *Params, name string, typ reflect.Type) reflect.Value {
	// Collect an array of slice elements with their indexes (and the max index).
	maxIndex := -1
	numNoIndex := 0
	sliceValues := []sliceValue{}

	// Factor out the common slice logic (between form values and files).
	processElement := func(key string, vals []string, files []*multipart.FileHeader) {
		if !strings.HasPrefix(key, name+"[") {
			return
		}

		// Extract the index, and the index where a sub-key starts. (e.g. field[0].subkey)
		index := -1
		leftBracket, rightBracket := len(name), strings.Index(key[len(name):], "]")+len(name)
		if rightBracket > leftBracket+1 {
			index, _ = strconv.Atoi(key[leftBracket+1 : rightBracket])
		}
		subKeyIndex := rightBracket + 1

		// Handle the indexed case.
		if index > -1 {
			if index > maxIndex {
				maxIndex = index
			}
			sliceValues = append(sliceValues, sliceValue{
				index: index,
				value: Bind(params, key[:subKeyIndex], typ.Elem()),
			})
			return
		}

		// It's an un-indexed element.  (e.g. element[])
		numNoIndex += len(vals) + len(files)
		for _, val := range vals {
			// Unindexed values can only be direct-bound.
			sliceValues = append(sliceValues, sliceValue{
				index: -1,
				value: BindValue(val, typ.Elem()),
			})
		}

		for _, fileHeader := range files {
			sliceValues = append(sliceValues, sliceValue{
				index: -1,
				value: BindFile(fileHeader, typ.Elem()),
			})
		}
	}

	for key, vals := range params.Values {
		processElement(key, vals, nil)
	}
	for key, fileHeaders := range params.Files {
		processElement(key, nil, fileHeaders)
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

// Break on dots and brackets.
// e.g. bar => "bar", bar.baz => "bar", bar[0] => "bar"
func nextKey(key string) string {
	fieldLen := strings.IndexAny(key, ".[")
	if fieldLen == -1 {
		return key
	}
	return key[:fieldLen]
}

func unbindSlice(output map[string]string, name string, val interface{}) {
	v := reflect.ValueOf(val)
	for i := 0; i < v.Len(); i++ {
		Unbind(output, fmt.Sprintf("%s[%d]", name, i), v.Index(i).Interface())
	}
}

func bindStruct(params *Params, name string, typ reflect.Type) reflect.Value {
	result := reflect.New(typ).Elem()
	fieldValues := make(map[string]reflect.Value)
	for key, _ := range params.Values {
		if !strings.HasPrefix(key, name+".") {
			continue
		}

		// Get the name of the struct property.
		// Strip off the prefix. e.g. foo.bar.baz => bar.baz
		suffix := key[len(name)+1:]
		fieldName := nextKey(suffix)
		fieldLen := len(fieldName)

		if _, ok := fieldValues[fieldName]; !ok {
			// Time to bind this field.  Get it and make sure we can set it.
			fieldValue := result.FieldByName(fieldName)
			if !fieldValue.IsValid() {
				WARN.Println("W: bindStruct: Field not found:", fieldName)
				continue
			}
			if !fieldValue.CanSet() {
				WARN.Println("W: bindStruct: Field not settable:", fieldName)
				continue
			}
			boundVal := Bind(params, key[:len(name)+1+fieldLen], fieldValue.Type())
			fieldValue.Set(boundVal)
			fieldValues[fieldName] = boundVal
		}
	}

	return result
}

func unbindStruct(output map[string]string, name string, iface interface{}) {
	val := reflect.ValueOf(iface)
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		structField := typ.Field(i)
		fieldValue := val.Field(i)

		// PkgPath is specified to be empty exactly for exported fields.
		if structField.PkgPath == "" {
			Unbind(output, fmt.Sprintf("%s.%s", name, structField.Name), fieldValue.Interface())
		}
	}
}

// Helper that returns an upload of the given name, or nil.
func getMultipartFile(params *Params, name string) multipart.File {
	for _, fileHeader := range params.Files[name] {
		file, err := fileHeader.Open()
		if err == nil {
			return file
		}
		WARN.Println("Failed to open uploaded file", name, ":", err)
	}
	return nil
}

func bindFile(params *Params, name string, typ reflect.Type) reflect.Value {
	reader := getMultipartFile(params, name)
	if reader == nil {
		return reflect.Zero(typ)
	}

	// If it's already stored in a temp file, just return that.
	if osFile, ok := reader.(*os.File); ok {
		return reflect.ValueOf(osFile)
	}

	// Otherwise, have to store it.
	tmpFile, err := ioutil.TempFile("", "revel-upload")
	if err != nil {
		WARN.Println("Failed to create a temp file to store upload:", err)
		return reflect.Zero(typ)
	}

	// Register it to be deleted after the request is done.
	params.tmpFiles = append(params.tmpFiles, tmpFile)

	_, err = io.Copy(tmpFile, reader)
	if err != nil {
		WARN.Println("Failed to copy upload to temp file:", err)
		return reflect.Zero(typ)
	}

	_, err = tmpFile.Seek(0, 0)
	if err != nil {
		WARN.Println("Failed to seek to beginning of temp file:", err)
		return reflect.Zero(typ)
	}

	return reflect.ValueOf(tmpFile)
}

func bindByteArray(params *Params, name string, typ reflect.Type) reflect.Value {
	if reader := getMultipartFile(params, name); reader != nil {
		b, err := ioutil.ReadAll(reader)
		if err == nil {
			return reflect.ValueOf(b)
		}
		WARN.Println("Error reading uploaded file contents:", err)
	}
	return reflect.Zero(typ)
}

func bindReadSeeker(params *Params, name string, typ reflect.Type) reflect.Value {
	if reader := getMultipartFile(params, name); reader != nil {
		return reflect.ValueOf(reader.(io.ReadSeeker))
	}
	return reflect.Zero(typ)
}

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

// Bind takes the name and type of the desired parameter and constructs it
// from one or more values from Params.
// Returns the zero value of the type upon any sort of failure.
func Bind(params *Params, name string, typ reflect.Type) reflect.Value {
	if binder, found := binderForType(typ); found {
		return binder.Bind(params, name, typ)
	}
	return reflect.Zero(typ)
}

func BindValue(val string, typ reflect.Type) reflect.Value {
	return Bind(&Params{Values: map[string][]string{"": {val}}}, "", typ)
}

func BindFile(fileHeader *multipart.FileHeader, typ reflect.Type) reflect.Value {
	return Bind(&Params{Files: map[string][]*multipart.FileHeader{"": {fileHeader}}}, "", typ)
}

func Unbind(output map[string]string, name string, val interface{}) {
	if binder, found := binderForType(reflect.TypeOf(val)); found {
		if binder.Unbind != nil {
			binder.Unbind(output, name, val)
		} else {
			ERROR.Printf("revel/binder: can not unbind %s=%s", name, val)
		}
	}
}

func binderForType(typ reflect.Type) (Binder, bool) {
	binder, ok := TypeBinders[typ]
	if !ok {
		binder, ok = KindBinders[typ.Kind()]
		if !ok {
			WARN.Println("revel/binder: no binder for type:", typ)
			return Binder{}, false
		}
	}
	return binder, true
}
