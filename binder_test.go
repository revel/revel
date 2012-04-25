package play

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

type binderTestCaseArgs struct {
	name   string
	typeof interface{}
	kv     []keyValue
}

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
	testDate     = time.Date(1982, time.July, 9, 0, 0, 0, 0, time.UTC)
	testDatetime = time.Date(1982, time.July, 9, 21, 30, 0, 0, time.UTC)
)

var binderTestCases = map[*binderTestCaseArgs]interface{}{
	&binderTestCaseArgs{"int", 0, []keyValue{{"", "1"}}}:                                 1,
	&binderTestCaseArgs{"str", "", []keyValue{{"", "hello"}}}:                            "hello",
	&binderTestCaseArgs{"bool", true, []keyValue{{"", "true"}}}:                          true,
	&binderTestCaseArgs{"bool", true, []keyValue{{"", "1"}}}:                             true,
	&binderTestCaseArgs{"bool", true, []keyValue{{"", "on"}}}:                            true,
	&binderTestCaseArgs{"bool", false, []keyValue{{"", "false"}}}:                        false,
	&binderTestCaseArgs{"bool", false, []keyValue{{"", "0"}}}:                            false,
	&binderTestCaseArgs{"bool", false, []keyValue{{"", ""}}}:                             false,
	&binderTestCaseArgs{"date", time.Time{}, []keyValue{{"", "1982-07-09"}}}:             testDate,
	&binderTestCaseArgs{"dt", time.Time{}, []keyValue{{"", "1982-07-09 21:30"}}}:         testDatetime,
	&binderTestCaseArgs{"arr", []int{}, []keyValue{{"arr[0]", "1"}}}:                     []int{1},
	&binderTestCaseArgs{"uarr", []int{}, []keyValue{{"arr[]", "1"}}}:                     []int{1},
	&binderTestCaseArgs{"arruarr", [][]int{{}}, []keyValue{{"arr[0][]", "1"}}}:           [][]int{{1}},
	&binderTestCaseArgs{"uarrarr", [][]int{{}}, []keyValue{{"arr[][0]", "1"}}}:           [][]int{{1}},
	&binderTestCaseArgs{"uarruarr", [][]int{{}}, []keyValue{{"arr[][]", "1"}}}:           [][]int{{1}},
	&binderTestCaseArgs{"uarruarrstr", [][]string{{""}}, []keyValue{{"arr[][]", "foo"}}}: [][]string{{"foo"}},
	&binderTestCaseArgs{"2darr", [][]int{{}}, []keyValue{
		{"arr[0][0]", "0"}, {"arr[0][1]", "1"},
		{"arr[1][0]", "10"}, {"arr[1][1]", "11"}},
	}: [][]int{{0, 1}, {10, 11}},
	&binderTestCaseArgs{"simple struct", A{}, []keyValue{
		{"a.Id", "123"}, {"a.Name", "rob"}},
	}: A{Id: 123, Name: "rob"},
	&binderTestCaseArgs{"double struct", A{}, []keyValue{
		{"a.Id", "123"}, {"a.Name", "rob"}, {"a.B.Extra", "hello"}},
	}: A{Id: 123, Name: "rob", B: B{Extra: "hello"}},
	&binderTestCaseArgs{"pointer to struct", &A{}, []keyValue{
		{"a.Id", "123"}, {"a.Name", "rob"}, {"a.B.Extra", "hello"}},
	}: &A{Id: 123, Name: "rob", B: B{Extra: "hello"}},

	// This one uses a custom-added TimeFormat
	&binderTestCaseArgs{"customDate", time.Time{}, []keyValue{{"", "07/09/1982"}}}: testDate,

	// TODO: Tests that use TypeBinders

	// Invalid value tests (the result should always be the zero value for that type)
	// The point of these is to ensure that invalid user input does not cause panics.
	&binderTestCaseArgs{"invalid int", 0, []keyValue{{"", "xyz"}}}:       0,
	&binderTestCaseArgs{"invalid int 2", 0, []keyValue{{"", ""}}}:        0,
	&binderTestCaseArgs{"invalid bool", false, []keyValue{{"", "xyz"}}}:  false,
	&binderTestCaseArgs{"invalid arr", []int{}, []keyValue{{"", "xyz"}}}: []int{},
	&binderTestCaseArgs{"private field", A{}, []keyValue{
		{"a.private", "123"}},
	}: A{},
}

func init() {
	TimeFormats = append(TimeFormats, "01/02/2006")
}

func TestBinder(t *testing.T) {
	for k, v := range binderTestCases {
		actual := Bind(reflect.TypeOf(k.typeof), k.kv)
		expected := reflect.ValueOf(v)
		valEq(t, k.name, actual, expected)
	}
}

type bindKeyTestCaseArgs struct {
	name   string
	typeof interface{}
	key    string
	params map[string][]string
}

var bindKeyTestCases = map[*bindKeyTestCaseArgs]interface{}{
	&bindKeyTestCaseArgs{"int", 0, "id", map[string][]string{"id": {"5"}}}:                      5,
	&bindKeyTestCaseArgs{"int2", 0, "id", map[string][]string{"id": {"5"}, "name": {"rob"}}}:    5,
	&bindKeyTestCaseArgs{"str2", "", "name", map[string][]string{"id": {"5"}, "name": {"rob"}}}: "rob",
	&bindKeyTestCaseArgs{"struct", A{}, "a", map[string][]string{
		"a.Id":      {"5"},
		"a.Name":    {"rob"},
		"alpha":     {"bar"},
		"a.B.Extra": {"foo"},
	}}: A{Id: 5, Name: "rob", B: B{"foo"}},
	&bindKeyTestCaseArgs{"array-struct", []A{}, "arr", map[string][]string{
		"arr[0].Id":      {"5"},
		"arr[0].Name":    {"rob"},
		"arr[0].B.Extra": {"foo"},
		"arrgh":          {"bar"},
		"arr[1].Id":      {"8"},
		"arr[1].Name":    {"bill"},
	}}: []A{
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
}

func TestBindKey(t *testing.T) {
	for k, v := range bindKeyTestCases {
		actual := BindKey(k.params, reflect.TypeOf(k.typeof), k.key)
		expected := reflect.ValueOf(v)
		valEq(t, k.name, actual, expected)
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
			t.Errorf("(actual) %s != %s (expected)", actual, expected)
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
