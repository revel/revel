// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Test that the render response is as expected.
func TestBenchmarkRender(t *testing.T) {
	startFakeBookingApp()
	resp := httptest.NewRecorder()
	c := NewController(NewRequest(showRequest), NewResponse(resp))
	if err := c.SetAction("Hotels", "Show"); err != nil {
		t.Errorf("SetAction failed: %s", err)
	}
	result := Hotels{c}.Show(3)
	result.Apply(c.Request, c.Response)
	if !strings.Contains(resp.Body.String(), "300 Main St.") {
		t.Errorf("Failed to find hotel address in action response:\n%s", resp.Body)
	}
}

func BenchmarkRenderChunked(b *testing.B) {
	startFakeBookingApp()
	resp := httptest.NewRecorder()
	resp.Body = nil
	c := NewController(NewRequest(showRequest), NewResponse(resp))
	if err := c.SetAction("Hotels", "Show"); err != nil {
		b.Errorf("SetAction failed: %s", err)
	}
	Config.SetOption("results.chunked", "true")
	b.ResetTimer()

	hotels := Hotels{c}
	for i := 0; i < b.N; i++ {
		hotels.Show(3).Apply(c.Request, c.Response)
	}
}

func BenchmarkRenderNotChunked(b *testing.B) {
	startFakeBookingApp()
	resp := httptest.NewRecorder()
	resp.Body = nil
	c := NewController(NewRequest(showRequest), NewResponse(resp))
	if err := c.SetAction("Hotels", "Show"); err != nil {
		b.Errorf("SetAction failed: %s", err)
	}
	Config.SetOption("results.chunked", "false")
	b.ResetTimer()

	hotels := Hotels{c}
	for i := 0; i < b.N; i++ {
		hotels.Show(3).Apply(c.Request, c.Response)
	}
}
