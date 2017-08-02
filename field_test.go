// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"reflect"
	"testing"
)

type H map[string]interface{}

func TestField(t *testing.T) {

	for _, test := range []struct {
		excepted   string
		name       string
		renderArgs map[string]interface{}
	}{
		{
			excepted:   "a",
			name:       "f1[a]",
			renderArgs: map[string]interface{}{"f1": H{"a": "a"}},
		},
		{
			excepted:   "b",
			name:       "f1[a].b",
			renderArgs: map[string]interface{}{"f1": H{"a": H{"b": "b"}}},
		},
	} {
		test.renderArgs["errors"] = map[string]*ValidationError{}
		field := NewField(test.name, test.renderArgs)
		actual := field.Value()

		if !reflect.DeepEqual(test.excepted, actual) {
			t.Error(test.name, "except", test.excepted, ", actual is", actual)
		}
	}
}
