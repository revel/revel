package revel

import (
	"reflect"
	"testing"
)

func TestTplEq(t *testing.T) {
	testRow := func(t *testing.T, row, row2 []interface{}, expected bool) {
		for _, a := range row {
			for _, b := range row2 {
				ok := tplEq(a, b)
				if ok != expected {
					ak := reflect.TypeOf(a).Kind()
					bk := reflect.TypeOf(b).Kind()
					t.Errorf("eq(%s=%v,%s=%v) want %t got %t", ak, a, bk, b, expected, ok)
				}
			}
		}
	}
	tm := make(map[string][]interface{})
	type testStruct struct{}
	type testStruct2 struct{}
	i, i2 := 8, 9
	s, s2 := "@æœ•µ\n\tüöäß", "@æœ•µ\n\tüöäss"
	slice, slice2 := []int{1, 2, 3, 4, 5}, []int{1, 2, 3, 4, 5}
	slice3, slice4 := []int{5, 4, 3, 2, 1}, []int{5, 4, 3, 2, 1}

	tm["slices"] = []interface{}{slice, slice2}
	tm["slices2"] = []interface{}{slice3, slice4}
	tm["types"] = []interface{}{new(testStruct), new(testStruct)}
	tm["types2"] = []interface{}{new(testStruct2), new(testStruct2)}
	tm["ints"] = []interface{}{int(i), int8(i), int16(i), int32(i), int64(i)}
	tm["ints2"] = []interface{}{int(i2), int8(i2), int16(i2), int32(i2), int64(i2)}
	tm["uints"] = []interface{}{uint(i), uint8(i), uint16(i), uint32(i), uint64(i)}
	tm["uints2"] = []interface{}{uint(i2), uint8(i2), uint16(i2), uint32(i2), uint64(i2)}
	tm["floats"] = []interface{}{float32(i), float64(i)}
	tm["floats2"] = []interface{}{float32(i2), float64(i2)}
	tm["strings"] = []interface{}{[]byte(s), s}
	tm["strings2"] = []interface{}{[]byte(s2), s2}

	testRow(t, tm["slices"], tm["slices"], true)
	testRow(t, tm["slices"], tm["slices2"], false)
	testRow(t, tm["slices2"], tm["slices"], false)

	testRow(t, tm["types"], tm["types"], true)
	testRow(t, tm["types2"], tm["types"], false)
	testRow(t, tm["types"], tm["types2"], false)

	testRow(t, tm["ints"], tm["ints"], true)
	testRow(t, tm["ints"], tm["ints2"], false)
	testRow(t, tm["ints2"], tm["ints"], false)

	testRow(t, tm["uints"], tm["uints"], true)
	testRow(t, tm["uints2"], tm["uints"], false)
	testRow(t, tm["uints"], tm["uints2"], false)

	testRow(t, tm["floats"], tm["floats"], true)
	testRow(t, tm["floats2"], tm["floats"], false)
	testRow(t, tm["floats"], tm["floats2"], false)

	testRow(t, tm["strings"], tm["strings"], true)
	testRow(t, tm["strings2"], tm["strings"], false)
	testRow(t, tm["strings"], tm["strings2"], false)
}
func BenchmarkEqFunction(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tplEq([]byte("Hello You"), "Hello You")
	}
}
