package revel

import (
	"path"
	"path/filepath"
	"reflect"
	"testing"
)

func TestContentTypeByFilename(t *testing.T) {
	testCases := map[string]string{
		"xyz.jpg":       "image/jpeg",
		"helloworld.c":  "text/x-c; charset=utf-8",
		"helloworld.":   "application/octet-stream",
		"helloworld":    "application/octet-stream",
		"hello.world.c": "text/x-c; charset=utf-8",
	}
	srcPath, _ := findSrcPaths(REVEL_IMPORT_PATH)
	ConfPaths = []string{path.Join(
		srcPath,
		filepath.FromSlash(REVEL_IMPORT_PATH),
		"conf"),
	}
	LoadMimeConfig()
	for filename, expected := range testCases {
		actual := ContentTypeByFilename(filename)
		if actual != expected {
			t.Errorf("%s: %s, Expected %s", filename, actual, expected)
		}
	}
}

func TestEqual(t *testing.T) {
	type testStruct struct{}
	type testStruct2 struct{}
	i, i2 := 8, 9
	s, s2 := "@æœ•µ\n\tüöäß", "@æœ•µ\n\tüöäss"
	slice, slice2 := []int{1, 2, 3, 4, 5}, []int{1, 2, 3, 4, 5}
	slice3, slice4 := []int{5, 4, 3, 2, 1}, []int{5, 4, 3, 2, 1}

	tm := map[string][]interface{}{
		"slices":   {slice, slice2},
		"slices2":  {slice3, slice4},
		"types":    {new(testStruct), new(testStruct)},
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

	testRow := func(row, row2 string, expected bool) {
		for _, a := range tm[row] {
			for _, b := range tm[row2] {
				ok := Equal(a, b)
				if ok != expected {
					ak := reflect.TypeOf(a).Kind()
					bk := reflect.TypeOf(b).Kind()
					t.Errorf("eq(%s=%v,%s=%v) want %t got %t", ak, a, bk, b, expected, ok)
				}
			}
		}
	}

	testRow("slices", "slices", true)
	testRow("slices", "slices2", false)
	testRow("slices2", "slices", false)

	testRow("types", "types", true)
	testRow("types2", "types", false)
	testRow("types", "types2", false)

	testRow("ints", "ints", true)
	testRow("ints", "ints2", false)
	testRow("ints2", "ints", false)

	testRow("uints", "uints", true)
	testRow("uints2", "uints", false)
	testRow("uints", "uints2", false)

	testRow("floats", "floats", true)
	testRow("floats2", "floats", false)
	testRow("floats", "floats2", false)

	testRow("strings", "strings", true)
	testRow("strings2", "strings", false)
	testRow("strings", "strings2", false)
}
