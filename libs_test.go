// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import "testing"

func TestToBooleanForFalse(t *testing.T) {
	if ToBool(nil) ||
		ToBool([]string{}) ||
		ToBool(map[string]string{}) ||
		ToBool(0) ||
		ToBool(0.0) ||
		ToBool("") ||
		ToBool("false") ||
		ToBool("0") ||
		ToBool("0.0") ||
		ToBool("off") ||
		ToBool("f") {
		t.Error("Expected 'false' got 'true'")
	}
}

func TestToBooleanForTrue(t *testing.T) {
	if !ToBool([]string{"true"}) ||
		!ToBool(map[string]string{"true": "value"}) ||
		!ToBool(1) ||
		!ToBool(0.1) ||
		!ToBool("not empty") ||
		!ToBool("true") ||
		!ToBool("1") ||
		!ToBool("1.0") ||
		!ToBool("on") ||
		!ToBool("t") {
		t.Error("Expected 'true' got 'false'")
	}
}
