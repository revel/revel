// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"reflect"
	"testing"
)

type Struct1 struct {
	X int
}

func (s Struct1) Method1() {}

type Interface1 interface {
	Method1()
}

var (
	struct1                    = Struct1{1}
	ptrStruct                  = &Struct1{2}
	emptyIface     interface{} = Struct1{3}
	iface1         Interface1  = Struct1{4}
	sliceStruct                = []Struct1{{5}, {6}, {7}}
	ptrSliceStruct             = []*Struct1{{8}, {9}, {10}}

	valueMap = map[string]interface{}{
		"bytes":          []byte{0x61, 0x62, 0x63, 0x64},
		"string":         "string",
		"bool":           true,
		"int":            5,
		"int8":           int8(5),
		"int16":          int16(5),
		"int32":          int32(5),
		"int64":          int64(5),
		"uint":           uint(5),
		"uint8":          uint8(5),
		"uint16":         uint16(5),
		"uint32":         uint32(5),
		"uint64":         uint64(5),
		"float32":        float32(5),
		"float64":        float64(5),
		"array":          [5]int{1, 2, 3, 4, 5},
		"slice":          []int{1, 2, 3, 4, 5},
		"emptyIf":        emptyIface,
		"Iface1":         iface1,
		"map":            map[string]string{"foo": "bar"},
		"ptrStruct":      ptrStruct,
		"struct1":        struct1,
		"sliceStruct":    sliceStruct,
		"ptrSliceStruct": ptrSliceStruct,
	}
)

// Test passing all kinds of data between serialize and deserialize.
func TestRoundTrip(t *testing.T) {
	for _, expected := range valueMap {
		bytes, err := Serialize(expected)
		if err != nil {
			t.Error(err)
			continue
		}

		ptrActual := reflect.New(reflect.TypeOf(expected)).Interface()
		err = Deserialize(bytes, ptrActual)
		if err != nil {
			t.Error(err)
			continue
		}

		actual := reflect.ValueOf(ptrActual).Elem().Interface()
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("(expected) %T %v != %T %v (actual)", expected, expected, actual, actual)
		}
	}
}

func zeroMap(arg map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for key, value := range arg {
		result[key] = reflect.Zero(reflect.TypeOf(value)).Interface()
	}
	return result
}
