package play

import (
	"fmt"
	"regexp"
)

type ValidationError struct {
	Message, Key string
}

// Returns the Message.
func (e *ValidationError) String() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// A Validation context manages data validation and error messages.
type Validation struct {
	Errors []*ValidationError
	keep   bool
}

func (v *Validation) Keep() {
	v.keep = true
}

func (v *Validation) Clear() {
	v.Errors = []*ValidationError{}
}

func (v *Validation) HasErrors() bool {
	return len(v.Errors) > 0
}

// Return the errors mapped by key.
// If there are multiple validation errors associated with a single key, the
// first one "wins".  (Typically the first validation will be the more basic).
func (v *Validation) ErrorMap() map[string]*ValidationError {
	m := map[string]*ValidationError{}
	for _, e := range v.Errors {
		if _, ok := m[e.Key]; !ok {
			m[e.Key] = e
		}
	}
	return m
}

// A ValidationResult is returned from every validation method.
// It provides an indication of success, and a pointer to the Error (if any).
type ValidationResult struct {
	Error *ValidationError
	Ok    bool
}

func (r *ValidationResult) Key(key string) *ValidationResult {
	if r.Error != nil {
		r.Error.Key = key
	}
	return r
}

func (r *ValidationResult) Message(message string) *ValidationResult {
	if r.Error != nil {
		r.Error.Message = message
	}
	return r
}

type Check interface {
	IsSatisfied(interface{}) bool
	DefaultMessage() string
}

type Required struct{}

func (r Required) IsSatisfied(obj interface{}) bool {
	if obj == nil {
		return false
	}

	if str, ok := obj.(string); ok {
		return len(str) > 0
	}
	if list, ok := obj.([]interface{}); ok {
		return len(list) > 0
	}
	if b, ok := obj.(bool); ok {
		return b
	}
	if i, ok := obj.(int); ok {
		return i > 0
	}
	return true
}

func (r Required) DefaultMessage() string {
	return "Required"
}

// Test that the argument is non-nil and non-empty (if string or list)
func (v *Validation) Required(obj interface{}) *ValidationResult {
	return v.check(Required{}, obj)
}

type Min struct {
	min int
}

func (m Min) IsSatisfied(obj interface{}) bool {
	num, ok := obj.(int)
	if ok {
		return num >= m.min
	}
	return false
}

func (m Min) DefaultMessage() string {
	return fmt.Sprintln("Minimum is", m.min)
}

func (v *Validation) Min(n int, min int) *ValidationResult {
	return v.check(Min{min}, n)
}

type Max struct {
	max int
}

func (m Max) IsSatisfied(obj interface{}) bool {
	num, ok := obj.(int)
	if ok {
		return num <= m.max
	}
	return false
}

func (m Max) DefaultMessage() string {
	return fmt.Sprintln("Maximum is", m.max)
}

func (v *Validation) Max(n int, max int) *ValidationResult {
	return v.check(Max{max}, n)
}

// Requires an array or string to be at least a given length.
type MinSize struct {
	min int
}

func (m MinSize) IsSatisfied(obj interface{}) bool {
	if arr, ok := obj.([]interface{}); ok {
		return len(arr) >= m.min
	}
	if str, ok := obj.(string); ok {
		return len(str) >= m.min
	}
	return false
}

func (m MinSize) DefaultMessage() string {
	return fmt.Sprintln("Minimum size is", m.min)
}

func (v *Validation) MinSize(obj interface{}, min int) *ValidationResult {
	return v.check(MinSize{min}, obj)
}

// Requires an array or string to be at most a given length.
type MaxSize struct {
	max int
}

func (m MaxSize) IsSatisfied(obj interface{}) bool {
	if arr, ok := obj.([]interface{}); ok {
		return len(arr) <= m.max
	}
	if str, ok := obj.(string); ok {
		return len(str) <= m.max
	}
	return false
}

func (m MaxSize) DefaultMessage() string {
	return fmt.Sprintln("Maximum size is", m.max)
}

func (v *Validation) MaxSize(obj interface{}, max int) *ValidationResult {
	return v.check(MaxSize{max}, obj)
}

// Requires a string to match a given regex.
type Match struct {
	regex *regexp.Regexp
}

func (m Match) IsSatisfied(obj interface{}) bool {
	str := obj.(string)
	return m.regex.MatchString(str)
}

func (m Match) DefaultMessage() string {
	return fmt.Sprintln("Must match", m.regex)
}

func (v *Validation) Match(str string, regex *regexp.Regexp) *ValidationResult {
	return v.check(Match{regex}, str)
}

func (v *Validation) check(chk Check, obj interface{}) *ValidationResult {
	if chk.IsSatisfied(obj) {
		return &ValidationResult{Ok: true}
	}

	// Add the error to the validation context.
	err := &ValidationError{
		Message: chk.DefaultMessage(),
	}
	v.Errors = append(v.Errors, err)

	// Also return it in the result.
	return &ValidationResult{
		Ok:    false,
		Error: err,
	}
}
