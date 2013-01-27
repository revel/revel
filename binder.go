package rev

import (
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
type Binder func(params *Params, name string, typ reflect.Type) reflect.Value

// An adapter for easily making one-key-value binders.
func ValueBinder(f func(value string, typ reflect.Type) reflect.Value) Binder {
	return func(params *Params, name string, typ reflect.Type) reflect.Value {
		vals, ok := params.Values[name]
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
	KindBinders[reflect.Int] = ValueBinder(bindInt)
	KindBinders[reflect.Int8] = ValueBinder(bindInt8)
	KindBinders[reflect.Int16] = ValueBinder(bindInt16)
	KindBinders[reflect.Int32] = ValueBinder(bindInt32)
	KindBinders[reflect.Int64] = ValueBinder(bindInt64)

	KindBinders[reflect.String] = ValueBinder(bindStr)
	KindBinders[reflect.Bool] = ValueBinder(bindBool)
	KindBinders[reflect.Slice] = bindSlice
	KindBinders[reflect.Struct] = bindStruct
	KindBinders[reflect.Ptr] = bindPointer

	TypeBinders[reflect.TypeOf(time.Time{})] = ValueBinder(bindTime)

	// Uploads
	TypeBinders[reflect.TypeOf(&os.File{})] = bindFile
	TypeBinders[reflect.TypeOf([]byte{})] = bindByteArray
	TypeBinders[reflect.TypeOf((*io.Reader)(nil)).Elem()] = bindReadSeeker
	TypeBinders[reflect.TypeOf((*io.ReadSeeker)(nil)).Elem()] = bindReadSeeker
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
	return reflect.ValueOf(int(bindIntHelper(val, 0)))
}
func bindInt8(val string, typ reflect.Type) reflect.Value {
	return reflect.ValueOf(int8(bindIntHelper(val, 8)))
}
func bindInt16(val string, typ reflect.Type) reflect.Value {
	return reflect.ValueOf(int16(bindIntHelper(val, 16)))
}
func bindInt32(val string, typ reflect.Type) reflect.Value {
	return reflect.ValueOf(int32(bindIntHelper(val, 32)))
}
func bindInt64(val string, typ reflect.Type) reflect.Value {
	return reflect.ValueOf(int64(bindIntHelper(val, 64)))
}

func bindIntHelper(val string, bits int) int64 {
	if len(val) == 0 {
		return 0
	}
	intValue, err := strconv.ParseInt(val, 10, bits)
	if err != nil {
		WARN.Println(err)
		return 0
	}
	return intValue
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

func bindPointer(params *Params, name string, typ reflect.Type) reflect.Value {
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

func bindReader(params *Params, name string, typ reflect.Type) reflect.Value {
	if reader := getMultipartFile(params, name); reader != nil {
		return reflect.ValueOf(reader.(io.Reader))
	}
	return reflect.Zero(typ)
}

func bindReadSeeker(params *Params, name string, typ reflect.Type) reflect.Value {
	if reader := getMultipartFile(params, name); reader != nil {
		return reflect.ValueOf(reader.(io.ReadSeeker))
	}
	return reflect.Zero(typ)
}

// Parse the value string into a real Go value.
// Returns 0 values when things can not be parsed.
func Bind(params *Params, name string, typ reflect.Type) reflect.Value {
	if typ == nil {
		return reflect.ValueOf(nil)
	}

	binder, ok := TypeBinders[typ]
	if !ok {
		binder, ok = KindBinders[typ.Kind()]
		if !ok {
			WARN.Println("No binder for type:", typ)
			return reflect.Zero(typ)
		}
	}
	return binder(params, name, typ)
}

func BindValue(val string, typ reflect.Type) reflect.Value {
	return Bind(&Params{Values: map[string][]string{"": {val}}}, "", typ)
}

func BindFile(fileHeader *multipart.FileHeader, typ reflect.Type) reflect.Value {
	return Bind(&Params{Files: map[string][]*multipart.FileHeader{"": {fileHeader}}}, "", typ)
}
