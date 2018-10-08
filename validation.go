// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
)

// ValidationError simple struct to store the Message & Key of a validation error
type ValidationError struct {
	Message, Key string
}

// String returns the Message field of the ValidationError struct.
func (e *ValidationError) String() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Validation context manages data validation and error messages.
type Validation struct {
	Errors     []*ValidationError
	Request    *Request
	Translator func(locale, message string, args ...interface{}) string
	keep       bool
}

// Keep tells revel to set a flash cookie on the client to make the validation
// errors available for the next request.
// This is helpful  when redirecting the client after the validation failed.
// It is good practice to always redirect upon a HTTP POST request. Thus
// one should use this method when HTTP POST validation failed and redirect
// the user back to the form.
func (v *Validation) Keep() {
	v.keep = true
}

// Clear *all* ValidationErrors
func (v *Validation) Clear() {
	v.Errors = []*ValidationError{}
}

// HasErrors returns true if there are any (ie > 0) errors. False otherwise.
func (v *Validation) HasErrors() bool {
	return len(v.Errors) > 0
}

// ErrorMap returns the errors mapped by key.
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

// Error adds an error to the validation context.
func (v *Validation) Error(message string, args ...interface{}) *ValidationResult {
	result := v.ValidationResult(false).Message(message, args...)
	v.Errors = append(v.Errors, result.Error)
	return result
}

// Error adds an error to the validation context.
func (v *Validation) ErrorKey(message string, args ...interface{}) *ValidationResult {
	result := v.ValidationResult(false).MessageKey(message, args...)
	v.Errors = append(v.Errors, result.Error)
	return result
}

// Error adds an error to the validation context.
func (v *Validation) ValidationResult(ok bool) *ValidationResult {
	if ok {
		return &ValidationResult{Ok: ok}
	} else {
		return &ValidationResult{Ok: ok, Error: &ValidationError{}, Locale: v.Request.Locale, Translator: v.Translator}
	}
}

// ValidationResult is returned from every validation method.
// It provides an indication of success, and a pointer to the Error (if any).
type ValidationResult struct {
	Error      *ValidationError
	Ok         bool
	Locale     string
	Translator func(locale, message string, args ...interface{}) string
}

// Key sets the ValidationResult's Error "key" and returns itself for chaining
func (r *ValidationResult) Key(key string) *ValidationResult {
	if r.Error != nil {
		r.Error.Key = key
	}
	return r
}

// Message sets the error message for a ValidationResult. Returns itself to
// allow chaining.  Allows Sprintf() type calling with multiple parameters
func (r *ValidationResult) Message(message string, args ...interface{}) *ValidationResult {
	if r.Error != nil {
		if len(args) == 0 {
			r.Error.Message = message
		} else {
			r.Error.Message = fmt.Sprintf(message, args...)
		}
	}
	return r
}

// Allow a message key to be passed into the validation result. The Validation has already
// setup the translator to translate the message key
func (r *ValidationResult) MessageKey(message string, args ...interface{}) *ValidationResult {
	if r.Error == nil {
		return r
	}

	// If translator found, use that to create the message, otherwise call Message method
	if r.Translator != nil {
		r.Error.Message = r.Translator(r.Locale, message, args...)
	} else {
		r.Message(message, args...)
	}

	return r
}

// Required tests that the argument is non-nil and non-empty (if string or list)
func (v *Validation) Required(obj interface{}) *ValidationResult {
	return v.apply(Required{}, obj)
}

func (v *Validation) Min(n int, min int) *ValidationResult {
	return v.MinFloat(float64(n), float64(min))
}

func (v *Validation) MinFloat(n float64, min float64) *ValidationResult {
	return v.apply(Min{min}, n)
}

func (v *Validation) Max(n int, max int) *ValidationResult {
	return v.MaxFloat(float64(n), float64(max))
}

func (v *Validation) MaxFloat(n float64, max float64) *ValidationResult {
	return v.apply(Max{max}, n)
}

func (v *Validation) Range(n, min, max int) *ValidationResult {
	return v.RangeFloat(float64(n), float64(min), float64(max))
}

func (v *Validation) RangeFloat(n, min, max float64) *ValidationResult {
	return v.apply(Range{Min{min}, Max{max}}, n)
}

func (v *Validation) MinSize(obj interface{}, min int) *ValidationResult {
	return v.apply(MinSize{min}, obj)
}

func (v *Validation) MaxSize(obj interface{}, max int) *ValidationResult {
	return v.apply(MaxSize{max}, obj)
}

func (v *Validation) Length(obj interface{}, n int) *ValidationResult {
	return v.apply(Length{n}, obj)
}

func (v *Validation) Match(str string, regex *regexp.Regexp) *ValidationResult {
	return v.apply(Match{regex}, str)
}

func (v *Validation) Email(str string) *ValidationResult {
	return v.apply(Email{Match{emailPattern}}, str)
}

func (v *Validation) IPAddr(str string, cktype ...int) *ValidationResult {
	return v.apply(IPAddr{cktype}, str)
}

func (v *Validation) MacAddr(str string) *ValidationResult {
	return v.apply(IPAddr{}, str)
}

func (v *Validation) Domain(str string) *ValidationResult {
	return v.apply(Domain{}, str)
}

func (v *Validation) URL(str string) *ValidationResult {
	return v.apply(URL{}, str)
}

func (v *Validation) PureText(str string, m int) *ValidationResult {
	return v.apply(PureText{m}, str)
}

func (v *Validation) FilePath(str string, m int) *ValidationResult {
	return v.apply(FilePath{m}, str)
}

func (v *Validation) apply(chk Validator, obj interface{}) *ValidationResult {
	if chk.IsSatisfied(obj) {
		return v.ValidationResult(true)
	}

	// Get the default key.
	var key string
	if pc, _, line, ok := runtime.Caller(2); ok {
		f := runtime.FuncForPC(pc)
		if defaultKeys, ok := DefaultValidationKeys[f.Name()]; ok {
			key = defaultKeys[line]
		}
	} else {
		utilLog.Error("Validation: Failed to get Caller information to look up Validation key")
	}

	// Add the error to the validation context.
	err := &ValidationError{
		Message: chk.DefaultMessage(),
		Key:     key,
	}
	v.Errors = append(v.Errors, err)

	// Also return it in the result.
	vr := v.ValidationResult(false)
	vr.Error = err
	return vr
}

// Check applies a group of validators to a field, in order, and return the
// ValidationResult from the first one that fails, or the last one that
// succeeds.
func (v *Validation) Check(obj interface{}, checks ...Validator) *ValidationResult {
	var result *ValidationResult
	for _, check := range checks {
		result = v.apply(check, obj)
		if !result.Ok {
			return result
		}
	}
	return result
}

// ValidationFilter revel Filter function to be hooked into the filter chain.
func ValidationFilter(c *Controller, fc []Filter) {
	// If json request, we shall assume json response is intended,
	// as such no validation cookies should be tied response
	if c.Params != nil && c.Params.JSON != nil {
		c.Validation = &Validation{Request: c.Request, Translator: MessageFunc}
		fc[0](c, fc[1:])
	} else {
		errors, err := restoreValidationErrors(c.Request)
		c.Validation = &Validation{
			Errors:     errors,
			keep:       false,
			Request:    c.Request,
			Translator: MessageFunc,
		}
		hasCookie := (err != http.ErrNoCookie)

		fc[0](c, fc[1:])

		// Add Validation errors to ViewArgs.
		c.ViewArgs["errors"] = c.Validation.ErrorMap()

		// Store the Validation errors
		var errorsValue string
		if c.Validation.keep {
			for _, err := range c.Validation.Errors {
				if err.Message != "" {
					errorsValue += "\x00" + err.Key + ":" + err.Message + "\x00"
				}
			}
		}

		// When there are errors from Validation and Keep() has been called, store the
		// values in a cookie. If there previously was a cookie but no errors, remove
		// the cookie.
		if errorsValue != "" {
			c.SetCookie(&http.Cookie{
				Name:     CookiePrefix + "_ERRORS",
				Value:    url.QueryEscape(errorsValue),
				Domain:   CookieDomain,
				Path:     "/",
				HttpOnly: true,
				Secure:   CookieSecure,
			})
		} else if hasCookie {
			c.SetCookie(&http.Cookie{
				Name:     CookiePrefix + "_ERRORS",
				MaxAge:   -1,
				Domain:   CookieDomain,
				Path:     "/",
				HttpOnly: true,
				Secure:   CookieSecure,
			})
		}
	}
}

// Restore Validation.Errors from a request.
func restoreValidationErrors(req *Request) ([]*ValidationError, error) {
	var (
		err    error
		cookie ServerCookie
		errors = make([]*ValidationError, 0, 5)
	)
	if cookie, err = req.Cookie(CookiePrefix + "_ERRORS"); err == nil {
		ParseKeyValueCookie(cookie.GetValue(), func(key, val string) {
			errors = append(errors, &ValidationError{
				Key:     key,
				Message: val,
			})
		})
	}
	return errors, err
}

// DefaultValidationKeys register default validation keys for all calls to Controller.Validation.Func().
// Map from (package).func => (line => name of first arg to Validation func)
// E.g. "myapp/controllers.helper" or "myapp/controllers.(*Application).Action"
// This is set on initialization in the generated main.go file.
var DefaultValidationKeys map[string]map[int]string
