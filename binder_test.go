package play

import (
	"fmt"
	"reflect"
	"testing"
)

type binderTestCaseArgs struct {
	name string
	typeof interface{}
	kv []keyValue
}

var binderTestCases = map[*binderTestCaseArgs]interface{} {
	&binderTestCaseArgs{"int", 0, []keyValue{{"", "1"}}}: 1,
	&binderTestCaseArgs{"str", "", []keyValue{{"", "hello"}}}: "hello",
	&binderTestCaseArgs{"arr", []int{}, []keyValue{{"arr[0]", "1"}}}: []int{1},
	&binderTestCaseArgs{"uarr", []int{}, []keyValue{{"arr[]", "1"}}}: []int{1},
	&binderTestCaseArgs{"arruarr", [][]int{{}}, []keyValue{{"arr[0][]", "1"}}}: [][]int{{1}},
	&binderTestCaseArgs{"uarrarr", [][]int{{}}, []keyValue{{"arr[][0]", "1"}}}: [][]int{{1}},
	&binderTestCaseArgs{"uarruarr", [][]int{{}}, []keyValue{{"arr[][]", "1"}}}: [][]int{{1}},
	&binderTestCaseArgs{"uarruarrstr", [][]string{{""}}, []keyValue{{"arr[][]", "foo"}}}: [][]string{{"foo"}},
	&binderTestCaseArgs{"2darr", [][]int{{}}, []keyValue{
			{"arr[0][0]", "0"}, {"arr[0][1]", "1"},
			{"arr[1][0]", "10"}, {"arr[1][1]", "11"}},
	}: [][]int{{0, 1}, {10, 11}},
}

func TestBinder(t *testing.T) {
	for k, v := range binderTestCases {
		actual := Bind(reflect.TypeOf(k.typeof), k.kv)
		expected := reflect.ValueOf(v)
		valEq(t, k.name, actual, expected)
	}
}

func valEq(t *testing.T, name string, actual, expected reflect.Value) {
	switch expected.Kind() {
	case reflect.Slice:
		// Check the type/length/element type
		if ! eq(t, name + " (type)", actual.Kind(), expected.Kind()) ||
			! eq(t, name + " (len)", actual.Len(), expected.Len()) ||
			! eq(t, name + " (elem)", actual.Type().Elem(), expected.Type().Elem()) {
			t.Errorf("(actual) %s != %s (expected)", actual, expected)
			return
		}

		// Check value equality for each element.
		for i := 0; i < actual.Len(); i++ {
			valEq(t, fmt.Sprintf("%s[%d]", name, i), actual.Index(i), expected.Index(i))
		}
	default:
		eq(t, name, actual.Interface(), expected.Interface())
	}
}
