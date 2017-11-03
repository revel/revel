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
	viewArgs   map[string]interface{}
	controller *Controller
}

func NewField(name string, viewArgs map[string]interface{}) *Field {
	err, _ := viewArgs["errors"].(map[string]*ValidationError)[name]
	controller, _ := viewArgs["_controller"].(*Controller)
	return &Field{
		Name:       name,
		Error:      err,
		viewArgs:   viewArgs,
		controller: controller,
	}
}

// ID returns an identifier suitable for use as an HTML id.
func (f *Field) ID() string {
	return strings.Replace(f.Name, ".", "_", -1)
}

// Flash returns the flashed value of this Field.
func (f *Field) Flash() string {
	v, _ := f.viewArgs["flash"].(map[string]string)[f.Name]
	return v
}

// Options returns the option list of this Field.
func (f *Field) Options() []string {
	if f.viewArgs["options"] == nil {
		return nil
	}
	v, _ := f.viewArgs["options"].(map[string][]string)[f.Name]
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
	answer, ok := f.viewArgs[pieces[0]]
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
		if errorClass, ok := f.viewArgs["ERROR_CLASS"]; ok {
			return errorClass.(string)
		}
		return ErrorCSSClass
	}
	return ""
}

// Get the short name and translate it
func (f *Field) ShortName() string {
	name := f.Name
	if i := strings.LastIndex(name, "."); i > 0 {
		name = name[i+1:]
	}
	return f.Translate(name)
}

// Translate the text
func (f *Field) Translate(text string, args ...interface{}) string {
	if f.controller != nil {
		text = f.controller.Message(text, args...)
	}
	return text
}
