// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"fmt"
	"reflect"
	"regexp"
	"time"
	"unicode/utf8"
)

type Validator interface {
	IsSatisfied(interface{}) bool
	DefaultMessage() string
}

type Required struct{}

func ValidRequired() Required {
	return Required{}
}

func (r Required) IsSatisfied(obj interface{}) bool {
	if obj == nil {
		return false
	}

	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) > 0
	}
	if b, ok := obj.(bool); ok {
		return b
	}
	if i, ok := obj.(int); ok {
		return i != 0
	}
	if t, ok := obj.(time.Time); ok {
		return !t.IsZero()
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() > 0
	}
	return true
}

func (r Required) DefaultMessage() string {
	return "Required"
}

type Min struct {
	Min int
}

func ValidMin(min int) Min {
	return Min{min}
}

func (m Min) IsSatisfied(obj interface{}) bool {
	num, ok := obj.(int)
	if ok {
		return num >= m.Min
	}
	return false
}

func (m Min) DefaultMessage() string {
	return fmt.Sprintln("Minimum is", m.Min)
}

type Max struct {
	Max int
}

func ValidMax(max int) Max {
	return Max{max}
}

func (m Max) IsSatisfied(obj interface{}) bool {
	num, ok := obj.(int)
	if ok {
		return num <= m.Max
	}
	return false
}

func (m Max) DefaultMessage() string {
	return fmt.Sprintln("Maximum is", m.Max)
}

// Range requires an integer to be within Min, Max inclusive.
type Range struct {
	Min
	Max
}

func ValidRange(min, max int) Range {
	return Range{Min{min}, Max{max}}
}

func (r Range) IsSatisfied(obj interface{}) bool {
	return r.Min.IsSatisfied(obj) && r.Max.IsSatisfied(obj)
}

func (r Range) DefaultMessage() string {
	return fmt.Sprintln("Range is", r.Min.Min, "to", r.Max.Max)
}

// MinSize requires an array or string to be at least a given length.
type MinSize struct {
	Min int
}

func ValidMinSize(min int) MinSize {
	return MinSize{min}
}

func (m MinSize) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) >= m.Min
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() >= m.Min
	}
	return false
}

func (m MinSize) DefaultMessage() string {
	return fmt.Sprintln("Minimum size is", m.Min)
}

// MaxSize requires an array or string to be at most a given length.
type MaxSize struct {
	Max int
}

func ValidMaxSize(max int) MaxSize {
	return MaxSize{max}
}

func (m MaxSize) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) <= m.Max
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() <= m.Max
	}
	return false
}

func (m MaxSize) DefaultMessage() string {
	return fmt.Sprintln("Maximum size is", m.Max)
}

// Length requires an array or string to be exactly a given length.
type Length struct {
	N int
}

func ValidLength(n int) Length {
	return Length{n}
}

func (s Length) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) == s.N
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() == s.N
	}
	return false
}

func (s Length) DefaultMessage() string {
	return fmt.Sprintln("Required length is", s.N)
}

// Match requires a string to match a given regex.
type Match struct {
	Regexp *regexp.Regexp
}

func ValidMatch(regex *regexp.Regexp) Match {
	return Match{regex}
}

func (m Match) IsSatisfied(obj interface{}) bool {
	str := obj.(string)
	return m.Regexp.MatchString(str)
}

func (m Match) DefaultMessage() string {
	return fmt.Sprintln("Must match", m.Regexp)
}

var emailPattern = regexp.MustCompile("^[\\w!#$%&'*+/=?^_`{|}~-]+(?:\\.[\\w!#$%&'*+/=?^_`{|}~-]+)*@(?:[\\w](?:[\\w-]*[\\w])?\\.)+[a-zA-Z0-9](?:[\\w-]*[\\w])?$")

type Email struct {
	Match
}

func ValidEmail() Email {
	return Email{Match{emailPattern}}
}

func (e Email) DefaultMessage() string {
	return fmt.Sprintln("Must be a valid email address")
}
