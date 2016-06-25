// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"reflect"
	"strings"
)

// Sign a given string with the app-configured secret key.
// If no secret key is set, returns the empty string.
// Return the signature in base64 (URLEncoding).
func Sign(message string) string {
	if len(secretKey) == 0 {
		return ""
	}
	mac := hmac.New(sha1.New, secretKey)
	if _, err := io.WriteString(mac, message); err != nil {
		ERROR.Println("WriteString failed:", err)
		return ""
	}
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify returns true if the given signature is correct for the given message.
// e.g. it matches what we generate with Sign()
func Verify(message, sig string) bool {
	return hmac.Equal([]byte(sig), []byte(Sign(message)))
}

// ToBool method converts/assert value into true or false. Default is true.
// When converting to boolean, the following values are considered FALSE:
// - The integer value is 0 (zero)
// - The float value 0.0 (zero)
// - The complex value 0.0 (zero)
// - For string value, please refer `revel.Atob` method
// - An array, map, slice with zero elements
// - Boolean value returned as-is
// - "nil" value
func ToBool(val interface{}) bool {
	if val == nil {
		return false
	}

	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.String:
		return Atob(v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0.0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() != 0.0
	case reflect.Array, reflect.Map, reflect.Slice:
		return v.Len() != 0
	}

	// Return true by default
	return true
}

// Atob converts string into boolean. It is in-case sensitive
// When converting to boolean, the following values are considered FALSE:
// - The "" (empty) string
// - The "false" string
// - The "f" string
// - The "off" string,
// - The string "0" & "0.0"
func Atob(v string) bool {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "", "false", "off", "f", "0", "0.0":
		return false
	}

	// Return true by default
	return true
}
