package revel

import (
	"reflect"
	"testing"
)

func TestEq(t *testing.T) {
	f := func(t *testing.T, a, b interface{}, result bool) {
		eq := TemplateFuncs["eq"].(func(a, b interface{}) bool)
		ok := eq(a, b)
		ak := reflect.TypeOf(a).Kind()
		bk := reflect.TypeOf(b).Kind()
		if ok != result {
			t.Errorf("eq(%s=%v,%s=%v) want %t got %t", ak, a, bk, b, result, ok)
		}
	}
	i, i2 := 8, 9
	s, s2 := "@æœ•µ\n\tüöäß", "@æœ•µ\n\tüöäss"

	ints := [...]interface{}{int8(i), int16(i), int32(i), int64(i)}
	ints2 := [...]interface{}{int8(i2), int16(i2), int32(i2), int64(i2)}
	uints := [...]interface{}{uint8(i), uint16(i), uint32(i), uint64(i)}
	uints2 := [...]interface{}{uint8(i2), uint16(i2), uint32(i2), uint64(i2)}
	floats := [...]interface{}{float32(i), float64(i)}
	floats2 := [...]interface{}{float32(i2), float64(i2)}
	strings := [...]interface{}{[]byte(s), s}
	strings2 := [...]interface{}{[]byte(s2), s2}

	// ints against ints
	for _, a := range ints {
		for _, b := range ints {
			f(t, a, b, true)
		}
	}

	// ints against ints of diff value and vice versa
	for _, a := range ints {
		for _, b := range ints2 {
			f(t, a, b, false)
		}
	}
	for _, a := range ints2 {
		for _, b := range ints {
			f(t, a, b, false)
		}
	}

	// ints against uints and vice versa
	for _, a := range ints {
		for _, b := range uints {
			f(t, a, b, false)
		}
	}
	for _, a := range uints {
		for _, b := range ints {
			f(t, a, b, false)
		}
	}
	// ints against floats and vice versa
	for _, a := range ints {
		for _, b := range floats {
			f(t, a, b, false)
		}
	}
	for _, a := range floats {
		for _, b := range ints {
			f(t, a, b, false)
		}
	}
	// ints against strings and vice versa
	for _, a := range ints {
		for _, b := range strings {
			f(t, a, b, false)
		}
	}
	for _, a := range strings {
		for _, b := range ints {
			f(t, a, b, false)
		}
	}

	// uints vs uints
	for _, a := range uints {
		for _, b := range uints {
			f(t, a, b, true)
		}
	}

	// uints vs uints of other value and vice versa
	for _, a := range uints {
		for _, b := range uints2 {
			f(t, a, b, false)
		}
	}
	for _, a := range uints2 {
		for _, b := range uints {
			f(t, a, b, false)
		}
	}

	// uints vs floats and vice versa
	for _, a := range uints {
		for _, b := range floats {
			f(t, a, b, false)
		}
	}
	for _, a := range floats {
		for _, b := range uints {
			f(t, a, b, false)
		}
	}

	// uints vs strings and vice versa
	for _, a := range uints {
		for _, b := range strings {
			f(t, a, b, false)
		}
	}
	for _, a := range strings {
		for _, b := range uints {
			f(t, a, b, false)
		}
	}

	// floats vs floats
	for _, a := range floats {
		for _, b := range floats {
			f(t, a, b, true)
		}
	}

	// floats vs floats of other value and vice versa
	for _, a := range floats {
		for _, b := range floats2 {
			f(t, a, b, false)
		}
	}
	for _, a := range floats2 {
		for _, b := range floats {
			f(t, a, b, false)
		}
	}

	// floats vs strings and vice versa
	for _, a := range floats {
		for _, b := range strings {
			f(t, a, b, false)
		}
	}
	for _, a := range strings {
		for _, b := range floats {
			f(t, a, b, false)
		}
	}

	// strings vs strings
	for _, a := range strings {
		for _, b := range strings {
			f(t, a, b, true)
		}
	}
	// strings vs different strings
	for _, a := range strings {
		for _, b := range strings2 {
			f(t, a, b, false)
		}
	}
	for _, a := range strings2 {
		for _, b := range strings {
			f(t, a, b, false)
		}
	}
}
func BenchmarkEqFunction(b *testing.B) {
	b.StopTimer()
	eq := TemplateFuncs["eq"].(func(a, b interface{}) bool)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		eq([]byte("Hello You"), "Hello You")
	}
}
