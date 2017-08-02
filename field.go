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

// FlashArray returns the flashed value of this Field as a list split on comma.
func (f *Field) FlashArray() []string {
	v := f.Flash()
	if v == "" {
		return []string{}
	}
	return strings.Split(v, ",")
}

func readNext(nextKey string) (string, string) {
	switch nextKey[0] {
	case '[':
		idx := strings.IndexRune(nextKey, ']')
		if idx < 0 {
			return nextKey[1:], ""
		} else {
			return nextKey[1:idx], nextKey[idx+1:]
		}
	case '.':
		nextKey = nextKey[1:]
		fallthrough
	default:
		idx := strings.IndexAny(nextKey, ".[")
		if idx < 0 {
			return nextKey, ""
		} else if nextKey[idx] == '.' {
			return nextKey[:idx], nextKey[idx+1:]
		} else {
			return nextKey[:idx], nextKey[idx:]
		}
	}
}

// Value returns the current value of this Field.
func (f *Field) Value() interface{} {
	var fieldName string

	var nextKey = f.Name
	var val interface{} = f.viewArgs
	for nextKey != "" {
		fieldName, nextKey = readNext(nextKey)

		rVal := reflect.ValueOf(val)
		kind := rVal.Kind()
		if kind == reflect.Map {
			rFieldName := reflect.ValueOf(fieldName)
			val = rVal.MapIndex(rFieldName).Interface()
			if val == nil {
				return nil
			}
			continue
		}

		if kind == reflect.Ptr {
			rVal = rVal.Elem()
		}
		rVal = rVal.FieldByName(fieldName)
		if !rVal.IsValid() {
			return nil
		}
		val = rVal.Interface()
	}

	return val
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
