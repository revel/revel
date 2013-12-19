package revel

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Test that the render response is as expected.
func TestBenchmarkCompressed(t *testing.T) {
	startFakeBookingApp()
	resp := httptest.NewRecorder()
	c := NewController(NewRequest(showRequest), NewResponse(resp))
	c.SetAction("Hotels", "Show")
	Config.SetOption("results.compressed", "true")
	result := Hotels{c}.Show(3)
	result.Apply(c.Request, c.Response)
	if !strings.Contains(resp.Body.String(), "300 Main St.") {
		t.Errorf("Failed to find hotel address in action response:\n%s", resp.Body)
	}
}

func BenchmarkRenderCompressed(b *testing.B) {
	startFakeBookingApp()
	resp := httptest.NewRecorder()
	resp.Body = nil
	c := NewController(NewRequest(showRequest), NewResponse(resp))
	c.SetAction("Hotels", "Show")
	Config.SetOption("results.compressed", "true")
	b.ResetTimer()

	hotels := Hotels{c}
	for i := 0; i < b.N; i++ {
		hotels.Show(3).Apply(c.Request, c.Response)
	}
}

func BenchmarkRenderUnCompressed(b *testing.B) {
	startFakeBookingApp()
	resp := httptest.NewRecorder()
	resp.Body = nil
	c := NewController(NewRequest(showRequest), NewResponse(resp))
	c.SetAction("Hotels", "Show")
	Config.SetOption("results.compressed", "false")
	b.ResetTimer()

	hotels := Hotels{c}
	for i := 0; i < b.N; i++ {
		hotels.Show(3).Apply(c.Request, c.Response)
	}
}
