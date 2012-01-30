package play

// This file handles binding url parameters to data

import (
	"reflect"
	"strconv"
)

// Parse the value string into a real Go value.
// Returns 0 values when things can not be parsed.
func Bind(valueType reflect.Type, value string) reflect.Value {
	switch valueType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, _ := strconv.Atoi(value)
		return reflect.ValueOf(intValue)
	default:
		LOG.Println("No binder for type:", valueType)
	}
	return reflect.Zero(valueType)
}
