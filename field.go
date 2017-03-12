// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"reflect"
	"strings"
)

// Field represents a data field that may be collected in a web form.
type Field struct {
	Name       string
	Error      *ValidationError
	renderArgs map[string]interface{}
}

func NewField(name string, renderArgs map[string]interface{}) *Field {
	err, _ := renderArgs["errors"].(map[string]*ValidationError)[name]
	return &Field{
		Name:       name,
		Error:      err,
		renderArgs: renderArgs,
	}
}

// ID returns an identifier suitable for use as an HTML id.
func (f *Field) ID() string {
	return strings.Replace(f.Name, ".", "_", -1)
}

// Flash returns the flashed value of this Field.
func (f *Field) Flash() string {
	v, _ := f.renderArgs["flash"].(map[string]string)[f.Name]
	return v
}

// FlashArray returns the flashed value of this Field as a list split on comma.
func (f *Field) FlashArray() []string {
	v := f.Flash()
	if v == "" {
		return []string{}
	}
	return strings.Split(v, ",")
}

// Value returns the current value of this Field.
func (f *Field) Value() interface{} {
	pieces := strings.Split(f.Name, ".")
	answer, ok := f.renderArgs[pieces[0]]
	if !ok {
		return ""
	}

	val := reflect.ValueOf(answer)
	for i := 1; i < len(pieces); i++ {
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		val = val.FieldByName(pieces[i])
		if !val.IsValid() {
			return ""
		}
	}

	return val.Interface()
}

// ErrorClass returns ErrorCSSClass if this field has a validation error, else empty string.
func (f *Field) ErrorClass() string {
	if f.Error != nil {
		if errorClass, ok := f.renderArgs["ERROR_CLASS"]; ok {
			return errorClass.(string)
		}
		return ErrorCSSClass
	}
	return ""
}
