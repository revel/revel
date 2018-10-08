// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

type A struct {
	ID      int
	Name    string
	B       B
	private int
}

type B struct {
	Extra string
}

var (
	ParamTestValues = map[string][]string{
		"int":                            {"1"},
		"int8":                           {"1"},
		"int16":                          {"1"},
		"int32":                          {"1"},
		"int64":                          {"1"},
		"uint":                           {"1"},
		"uint8":                          {"1"},
		"uint16":                         {"1"},
		"uint32":                         {"1"},
		"uint64":                         {"1"},
		"float32":                        {"1.000000"},
		"float64":                        {"1.000000"},
		"str":                            {"hello"},
		"bool-true":                      {"true"},
		"bool-1":                         {"1"},
		"bool-on":                        {"on"},
		"bool-false":                     {"false"},
		"bool-0":                         {"0"},
		"bool-0.0":                       {"0.0"},
		"bool-off":                       {"off"},
		"bool-f":                         {"f"},
		"date":                           {"1982-07-09"},
		"datetime":                       {"1982-07-09 21:30"},
		"customDate":                     {"07/09/1982"},
		"arr[0]":                         {"1"},
		"arr[1]":                         {"2"},
		"arr[3]":                         {"3"},
		"uarr[]":                         {"1", "2"},
		"arruarr[0][]":                   {"1", "2"},
		"arruarr[1][]":                   {"3", "4"},
		"2darr[0][0]":                    {"0"},
		"2darr[0][1]":                    {"1"},
		"2darr[1][0]":                    {"10"},
		"2darr[1][1]":                    {"11"},
		"A.ID":                           {"123"},
		"A.Name":                         {"rob"},
		"B.ID":                           {"123"},
		"B.Name":                         {"rob"},
		"B.B.Extra":                      {"hello"},
		"pB.ID":                          {"123"},
		"pB.Name":                        {"rob"},
		"pB.B.Extra":                     {"hello"},
		"priv.private":                   {"123"},
		"arrC[0].ID":                     {"5"},
		"arrC[0].Name":                   {"rob"},
		"arrC[0].B.Extra":                {"foo"},
		"arrC[1].ID":                     {"8"},
		"arrC[1].Name":                   {"bill"},
		"m[a]":                           {"foo"},
		"m[b]":                           {"bar"},
		"m2[1]":                          {"foo"},
		"m2[2]":                          {"bar"},
		"m3[a]":                          {"1"},
		"m3[b]":                          {"2"},
		"m4[a].ID":                       {"1"},
		"m4[a].Name":                     {"foo"},
		"m4[b].ID":                       {"2"},
		"m4[b].Name":                     {"bar"},
		"mapWithAMuchLongerName[a].ID":   {"1"},
		"mapWithAMuchLongerName[a].Name": {"foo"},
		"mapWithAMuchLongerName[b].ID":   {"2"},
		"mapWithAMuchLongerName[b].Name": {"bar"},
		"invalidInt":                     {"xyz"},
		"invalidInt2":                    {""},
		"invalidBool":                    {"xyz"},
		"invalidArr":                     {"xyz"},
		"int8-overflow":                  {"1024"},
		"uint8-overflow":                 {"1024"},
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
	"uint":       1,
	"uint8":      uint8(1),
	"uint16":     uint16(1),
	"uint32":     uint32(1),
	"uint64":     uint64(1),
	"float32":    float32(1.0),
	"float64":    float64(1.0),
	"str":        "hello",
	"bool-true":  true,
	"bool-1":     true,
	"bool-on":    true,
	"bool-false": false,
	"bool-0":     false,
	"bool-0.0":   false,
	"bool-off":   false,
	"bool-f":     false,
	"date":       testDate,
	"datetime":   testDatetime,
	"customDate": testDate,
	"arr":        []int{1, 2, 0, 3},
	"uarr":       []int{1, 2},
	"arruarr":    [][]int{{1, 2}, {3, 4}},
	"2darr":      [][]int{{0, 1}, {10, 11}},
	"A":          A{ID: 123, Name: "rob"},
	"B":          A{ID: 123, Name: "rob", B: B{Extra: "hello"}},
	"pB":         &A{ID: 123, Name: "rob", B: B{Extra: "hello"}},
	"arrC": []A{
		{
			ID:   5,
			Name: "rob",
			B:    B{"foo"},
		},
		{
			ID:   8,
			Name: "bill",
		},
	},
	"m":  map[string]string{"a": "foo", "b": "bar"},
	"m2": map[int]string{1: "foo", 2: "bar"},
	"m3": map[string]int{"a": 1, "b": 2},
	"m4": map[string]A{"a": {ID: 1, Name: "foo"}, "b": {ID: 2, Name: "bar"}},

	// NOTE: We also include a map with a longer name than the others since this has caused problems
	// described in github issue #1285, resolved in pull request #1344. This test case should
	// prevent regression.
	"mapWithAMuchLongerName": map[string]A{"a": {ID: 1, Name: "foo"}, "b": {ID: 2, Name: "bar"}},

	// TODO: Tests that use TypeBinders

	// Invalid value tests (the result should always be the zero value for that type)
	// The point of these is to ensure that invalid user input does not cause panics.
	"invalidInt":     0,
	"invalidInt2":    0,
	"invalidBool":    true,
	"invalidArr":     []int{},
	"priv":           A{},
	"int8-overflow":  int8(0),
	"uint8-overflow": uint8(0),
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

func TestJsonBinder(t *testing.T) {
	// create a structure to be populated
	{
		d, _ := json.Marshal(map[string]int{"a": 1})
		params := &Params{JSON: d}
		foo := struct{ A int }{}
		c := NewTestController(nil, getMultipartRequest())

		ParseParams(params, NewRequest(c.Request.In))
		actual := Bind(params, "test", reflect.TypeOf(foo))
		valEq(t, "TestJsonBinder", reflect.ValueOf(actual.Interface().(struct{ A int }).A), reflect.ValueOf(1))
	}
	{
		d, _ := json.Marshal(map[string]interface{}{"a": map[string]int{"b": 45}})
		params := &Params{JSON: d}
		testMap := map[string]interface{}{}
		actual := Bind(params, "test", reflect.TypeOf(testMap)).Interface().(map[string]interface{})
		if actual["a"].(map[string]interface{})["b"].(float64) != 45 {
			t.Errorf("Failed to fetch map value %#v", actual["a"])
		}
		// Check to see if a named map works
		actualb := Bind(params, "test", reflect.TypeOf(map[string]map[string]float64{})).Interface().(map[string]map[string]float64)
		if actualb["a"]["b"] != 45 {
			t.Errorf("Failed to fetch map value %#v", actual["a"])
		}

	}
}

func TestBinder(t *testing.T) {
	// Reuse the mvc_test.go multipart request to test the binder.
	params := &Params{}
	c := NewTestController(nil, getMultipartRequest())
	ParseParams(params, NewRequest(c.Request.In))
	params.Values = ParamTestValues

	// Values
	for k, v := range binderTestCases {
		actual := Bind(params, k, reflect.TypeOf(v))
		expected := reflect.ValueOf(v)
		valEq(t, k, actual, expected)
	}

	// Files

	// Get the keys in sorted order to make the expectation right.
	keys := []string{}
	for k := range expectedFiles {
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
				for i := range fhs {
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

// Unbinding tests

var unbinderTestCases = map[string]interface{}{
	"int":        1,
	"int8":       int8(1),
	"int16":      int16(1),
	"int32":      int32(1),
	"int64":      int64(1),
	"uint":       1,
	"uint8":      uint8(1),
	"uint16":     uint16(1),
	"uint32":     uint32(1),
	"uint64":     uint64(1),
	"float32":    float32(1.0),
	"float64":    float64(1.0),
	"str":        "hello",
	"bool-true":  true,
	"bool-false": false,
	"date":       testDate,
	"datetime":   testDatetime,
	"arr":        []int{1, 2, 0, 3},
	"2darr":      [][]int{{0, 1}, {10, 11}},
	"A":          A{ID: 123, Name: "rob"},
	"B":          A{ID: 123, Name: "rob", B: B{Extra: "hello"}},
	"pB":         &A{ID: 123, Name: "rob", B: B{Extra: "hello"}},
	"arrC": []A{
		{
			ID:   5,
			Name: "rob",
			B:    B{"foo"},
		},
		{
			ID:   8,
			Name: "bill",
		},
	},
	"m":  map[string]string{"a": "foo", "b": "bar"},
	"m2": map[int]string{1: "foo", 2: "bar"},
	"m3": map[string]int{"a": 1, "b": 2},
}

// Some of the unbinding results are not exactly what is in ParamTestValues, since it
// serializes implicit zero values explicitly.
var unbinderOverrideAnswers = map[string]map[string]string{
	"arr": {
		"arr[0]": "1",
		"arr[1]": "2",
		"arr[2]": "0",
		"arr[3]": "3",
	},
	"A": {
		"A.ID":      "123",
		"A.Name":    "rob",
		"A.B.Extra": "",
	},
	"arrC": {
		"arrC[0].ID":      "5",
		"arrC[0].Name":    "rob",
		"arrC[0].B.Extra": "foo",
		"arrC[1].ID":      "8",
		"arrC[1].Name":    "bill",
		"arrC[1].B.Extra": "",
	},
	"m":  {"m[a]": "foo", "m[b]": "bar"},
	"m2": {"m2[1]": "foo", "m2[2]": "bar"},
	"m3": {"m3[a]": "1", "m3[b]": "2"},
}

func TestUnbinder(t *testing.T) {
	for k, v := range unbinderTestCases {
		actual := make(map[string]string)
		Unbind(actual, k, v)

		// Get the expected key/values.
		expected, ok := unbinderOverrideAnswers[k]
		if !ok {
			expected = make(map[string]string)
			for k2, v2 := range ParamTestValues {
				if k == k2 || strings.HasPrefix(k2, k+".") || strings.HasPrefix(k2, k+"[") {
					expected[k2] = v2[0]
				}
			}
		}

		// Compare length and values.
		if len(actual) != len(expected) {
			t.Errorf("Length mismatch\nExpected length %d, actual %d\nExpected: %s\nActual: %s",
				len(expected), len(actual), expected, actual)
		}
		for k, v := range actual {
			if expected[k] != v {
				t.Errorf("Value mismatch.\nExpected: %s\nActual: %s", expected, actual)
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
	case reflect.Map:
		if !eq(t, name+" (len)", actual.Len(), expected.Len()) {
			return
		}
		for _, key := range expected.MapKeys() {
			expectedValue := expected.MapIndex(key)
			actualValue := actual.MapIndex(key)
			if actualValue.IsValid() {
				valEq(t, fmt.Sprintf("%s[%s]", name, key), actualValue, expectedValue)
			} else {
				t.Errorf("Expected key %s not found", key)
			}
		}
	default:
		eq(t, name, actual.Interface(), expected.Interface())
	}
}

func init() {
	DateFormat = DefaultDateFormat
	DateTimeFormat = DefaultDateTimeFormat
	TimeFormats = append(TimeFormats, DefaultDateFormat, DefaultDateTimeFormat, "01/02/2006")
}
