package revel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test that a c.Render() returns an appropriate content type.
func TestAcceptsHeader(t *testing.T) {
	startFakeBookingApp()
	showRequest, _ := http.NewRequest("GET", "/hotels/3", nil)

	tests := []struct {
		accepts, prefix string
		status          int
	}{
		{"text/html", "<!DOCTYPE html>", 200},
		{"application/json", `{"HotelId":3`, 200},
		{"application/xml", "<Hotel><HotelId>3</HotelId>", 200},
		{"text/plain; charset=utf-8", "Not Found", 404},
	}

	for _, test := range tests {
		showRequest.Header.Set("Accept", test.accepts)
		resp := httptest.NewRecorder()
		handle(resp, showRequest)
		eq(t, "status code", resp.Code, test.status)
		if !strings.HasPrefix(resp.Body.String(), test.prefix) {
			t.Errorf("Unexpected body. Expected prefix %s, got:\n%s", test.prefix, resp.Body.String())
		}
		eq(t, "header", resp.Header().Get("Content-Type"), test.accepts)
	}
}
