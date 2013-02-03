package revel

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"
)

type A struct {
	Id      int
	Name    string
	B       B
	private int
}

type B struct {
	Extra string
}

var (
	PARAMS = map[string][]string{
		"int":             {"1"},
		"int8":            {"1"},
		"int16":           {"1"},
		"int32":           {"1"},
		"int64":           {"1"},
		"str":             {"hello"},
		"bool-true":       {"true"},
		"bool-1":          {"1"},
		"bool-on":         {"on"},
		"bool-false":      {"false"},
		"bool-0":          {"0"},
		"bool-off":        {""},
		"date":            {"1982-07-09"},
		"datetime":        {"1982-07-09 21:30"},
		"customDate":      {"07/09/1982"},
		"arr[0]":          {"1"},
		"arr[1]":          {"2"},
		"arr[3]":          {"3"},
		"uarr[]":          {"1", "2"},
		"arruarr[0][]":    {"1", "2"},
		"arruarr[1][]":    {"3", "4"},
		"2darr[0][0]":     {"0"},
		"2darr[0][1]":     {"1"},
		"2darr[1][0]":     {"10"},
		"2darr[1][1]":     {"11"},
		"A.Id":            {"123"},
		"A.Name":          {"rob"},
		"B.Id":            {"123"},
		"B.Name":          {"rob"},
		"B.B.Extra":       {"hello"},
		"pB.Id":           {"123"},
		"pB.Name":         {"rob"},
		"pB.B.Extra":      {"hello"},
		"priv.private":    {"123"},
		"arrC[0].Id":      {"5"},
		"arrC[0].Name":    {"rob"},
		"arrC[0].B.Extra": {"foo"},
		"arrC[1].Id":      {"8"},
		"arrC[1].Name":    {"bill"},
		"invalidInt":      {"xyz"},
		"invalidInt2":     {""},
		"invalidBool":     {"xyz"},
		"invalidArr":      {"xyz"},
	}

	testDate     = time.Date(1982, time.July, 9, 0, 0, 0, 0, time.UTC)
	testDatetime = time.Date(1982, time.July, 9, 21, 30, 0, 0, time.UTC)
)

var binderTestCases = map[string]interface{}{
	"int":        1,
	"int8":       int8(1),
	"int16":      int16(1),
	"int32":      int32(1),
	"int64":      int64(1),
	"str":        "hello",
	"bool-true":  true,
	"bool-1":     true,
	"bool-on":    true,
	"bool-false": false,
	"bool-0":     false,
	"bool-off":   false,
	"date":       testDate,
	"datetime":   testDatetime,
	"customDate": testDate,
	"arr":        []int{1, 2, 0, 3},
	"uarr":       []int{1, 2},
	"arruarr":    [][]int{{1, 2}, {3, 4}},
	"2darr":      [][]int{{0, 1}, {10, 11}},
	"A":          A{Id: 123, Name: "rob"},
	"B":          A{Id: 123, Name: "rob", B: B{Extra: "hello"}},
	"pB":         &A{Id: 123, Name: "rob", B: B{Extra: "hello"}},
	"arrC": []A{
		{
			Id:   5,
			Name: "rob",
			B:    B{"foo"},
		},
		{
			Id:   8,
			Name: "bill",
		},
	},

	// TODO: Tests that use TypeBinders

	// Invalid value tests (the result should always be the zero value for that type)
	// The point of these is to ensure that invalid user input does not cause panics.
	"invalidInt":  0,
	"invalidInt2": 0,
	"invalidBool": false,
	"invalidArr":  []int{},
	"priv":        A{},
}

func init() {
	TimeFormats = append(TimeFormats, DEFAULT_DATE_FORMAT, DEFAULT_DATETIME_FORMAT, "01/02/2006")
}

// Types that files may be bound to, and a func that can read the content from
// that type.
// TODO: Is there any way to create a slice, given only the element Type?
var fileBindings = []struct{ val, arrval, f interface{} }{
	{(**os.File)(nil), []*os.File{}, ioutil.ReadAll},
	{(*[]byte)(nil), [][]byte{}, func(b []byte) []byte { return b }},
	{(*io.Reader)(nil), []io.Reader{}, ioutil.ReadAll},
	{(*io.ReadSeeker)(nil), []io.ReadSeeker{}, ioutil.ReadAll},
}

func TestBinder(t *testing.T) {
	// Reuse the mvc_test.go multipart request to test the binder.
	params := ParseParams(NewRequest(getMultipartRequest()))
	params.Values = PARAMS

	// Values
	for k, v := range binderTestCases {
		actual := Bind(params, k, reflect.TypeOf(v))
		expected := reflect.ValueOf(v)
		valEq(t, k, actual, expected)
	}

	// Files

	// Get the keys in sorted order to make the expectation right.
	keys := []string{}
	for k, _ := range expectedFiles {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	expectedBoundFiles := make(map[string][]fh)
	for _, k := range keys {
		fhs := expectedFiles[k]
		k := nextKey(k)
		expectedBoundFiles[k] = append(expectedBoundFiles[k], fhs...)
	}

	for k, fhs := range expectedBoundFiles {

		if len(fhs) == 1 {
			// Test binding single files to: *os.File, []byte, io.Reader, io.ReadSeeker
			for _, binding := range fileBindings {
				typ := reflect.TypeOf(binding.val).Elem()
				actual := Bind(params, k, typ)
				if !actual.IsValid() || (actual.Kind() == reflect.Interface && actual.IsNil()) {
					t.Errorf("%s (%s) - Returned nil.", k, typ)
					continue
				}
				returns := reflect.ValueOf(binding.f).Call([]reflect.Value{actual})
				valEq(t, k, returns[0], reflect.ValueOf(fhs[0].content))
			}

		} else {
			// Test binding multi to:
			// []*os.File, [][]byte, []io.Reader, []io.ReadSeeker
			for _, binding := range fileBindings {
				typ := reflect.TypeOf(binding.arrval)
				actual := Bind(params, k, typ)
				if actual.Len() != len(fhs) {
					t.Fatalf("%s (%s) - Number of files: (expected) %d != %d (actual)",
						k, typ, len(fhs), actual.Len())
				}
				for i, _ := range fhs {
					returns := reflect.ValueOf(binding.f).Call([]reflect.Value{actual.Index(i)})
					if !returns[0].IsValid() {
						t.Errorf("%s (%s) - Returned nil.", k, typ)
						continue
					}
					valEq(t, k, returns[0], reflect.ValueOf(fhs[i].content))
				}
			}
		}
	}
}

// Helpers

func valEq(t *testing.T, name string, actual, expected reflect.Value) {
	switch expected.Kind() {
	case reflect.Slice:
		// Check the type/length/element type
		if !eq(t, name+" (type)", actual.Kind(), expected.Kind()) ||
			!eq(t, name+" (len)", actual.Len(), expected.Len()) ||
			!eq(t, name+" (elem)", actual.Type().Elem(), expected.Type().Elem()) {
			return
		}

		// Check value equality for each element.
		for i := 0; i < actual.Len(); i++ {
			valEq(t, fmt.Sprintf("%s[%d]", name, i), actual.Index(i), expected.Index(i))
		}
	case reflect.Ptr:
		// Check equality on the element type.
		valEq(t, name, actual.Elem(), expected.Elem())
	default:
		eq(t, name, actual.Interface(), expected.Interface())
	}
}
