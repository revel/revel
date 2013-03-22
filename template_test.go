package revel

import (
	"reflect"
	"testing"
)

func TestTplEq(t *testing.T) {
	type list []interface{}
	type testStruct struct{}
	type testStruct2 struct{}
	i, i2 := 8, 9
	s, s2 := "@æœ•µ\n\tüöäß", "@æœ•µ\n\tüöäss"
	slice, slice2 := []int{1, 2, 3, 4, 5}, []int{1, 2, 3, 4, 5}
	slice3, slice4 := []int{5, 4, 3, 2, 1}, []int{5, 4, 3, 2, 1}

	tm := map[string]list{
		"slices":   list{slice, slice2},
		"slices2":  list{slice3, slice4},
		"types":    list{new(testStruct), new(testStruct)},
		"types2":   {new(testStruct2), new(testStruct2)},
		"ints":     {int(i), int8(i), int16(i), int32(i), int64(i)},
		"ints2":    {int(i2), int8(i2), int16(i2), int32(i2), int64(i2)},
		"uints":    {uint(i), uint8(i), uint16(i), uint32(i), uint64(i)},
		"uints2":   {uint(i2), uint8(i2), uint16(i2), uint32(i2), uint64(i2)},
		"floats":   {float32(i), float64(i)},
		"floats2":  {float32(i2), float64(i2)},
		"strings":  {[]byte(s), s},
		"strings2": {[]byte(s2), s2},
	}

	testRow := func(row, row2 list, expected bool) {
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

	testRow(tm["slices"], tm["slices"], true)
	testRow(tm["slices"], tm["slices2"], false)
	testRow(tm["slices2"], tm["slices"], false)

	testRow(tm["types"], tm["types"], true)
	testRow(tm["types2"], tm["types"], false)
	testRow(tm["types"], tm["types2"], false)

	testRow(tm["ints"], tm["ints"], true)
	testRow(tm["ints"], tm["ints2"], false)
	testRow(tm["ints2"], tm["ints"], false)

	testRow(tm["uints"], tm["uints"], true)
	testRow(tm["uints2"], tm["uints"], false)
	testRow(tm["uints"], tm["uints2"], false)

	testRow(tm["floats"], tm["floats"], true)
	testRow(tm["floats2"], tm["floats"], false)
	testRow(tm["floats"], tm["floats2"], false)

	testRow(tm["strings"], tm["strings"], true)
	testRow(tm["strings2"], tm["strings"], false)
	testRow(tm["strings"], tm["strings2"], false)
}
