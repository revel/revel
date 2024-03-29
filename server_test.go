// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// This tries to benchmark the usual request-serving pipeline to get an overall
// performance metric.
//
// Each iteration runs one mock request to display a hotel's detail page by id.
//
// Contributing parts:
// - Routing
// - Controller lookup / invocation
// - Parameter binding
// - Session, flash, i18n cookies
// - Render() call magic
// - Template rendering.
func BenchmarkServeAction(b *testing.B) {
	benchmarkRequest(b, showRequest)
}

func BenchmarkServeJson(b *testing.B) {
	benchmarkRequest(b, jsonRequest)
}

func BenchmarkServePlaintext(b *testing.B) {
	benchmarkRequest(b, plaintextRequest)
}

// This tries to benchmark the static serving overhead when serving an "average
// size" 7k file.
func BenchmarkServeStatic(b *testing.B) {
	benchmarkRequest(b, staticRequest)
}

func benchmarkRequest(b *testing.B, req *http.Request) {
	startFakeBookingApp()
	b.ResetTimer()
	resp := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		CurrentEngine.(*GoHttpServer).Handle(resp, req)
	}
}

// Test that the booking app can be successfully run for a test.
func TestFakeServer(t *testing.T) {
	startFakeBookingApp()

	resp := httptest.NewRecorder()

	// First, test that the expected responses are actually generated
	CurrentEngine.(*GoHttpServer).Handle(resp, showRequest)
	if !strings.Contains(resp.Body.String(), "300 Main St.") {
		t.Errorf("Failed to find hotel address in action response:\n%s", resp.Body)
		t.FailNow()
	}
	resp.Body.Reset()

	CurrentEngine.(*GoHttpServer).Handle(resp, staticRequest)
	sessvarsSize := getFileSize(t, filepath.Join(BasePath, "public", "js", "sessvars.js"))
	if int64(resp.Body.Len()) != sessvarsSize {
		t.Errorf("Expected sessvars.js to have %d bytes, got %d:\n%s", sessvarsSize, resp.Body.Len(), resp.Body)
		t.FailNow()
	}
	resp.Body.Reset()

	CurrentEngine.(*GoHttpServer).Handle(resp, jsonRequest)
	if !strings.Contains(resp.Body.String(), `"Address":"300 Main St."`) {
		t.Errorf("Failed to find hotel address in JSON response:\n%s", resp.Body)
		t.FailNow()
	}
	resp.Body.Reset()

	CurrentEngine.(*GoHttpServer).Handle(resp, plaintextRequest)
	if resp.Body.String() != "Hello, World!" {
		t.Errorf("Failed to find greeting in plaintext response:\n%s", resp.Body)
		t.FailNow()
	}

	resp.Body = nil
}

func getFileSize(t *testing.T, name string) int64 {
	fi, err := os.Stat(name)
	if err != nil {
		t.Errorf("Unable to stat file:\n%s", name)
		t.FailNow()
	}
	return fi.Size()
}

// Ensure on app start runs in order.
func TestOnAppStart(t *testing.T) {
	str := ""
	a := assert.New(t)
	OnAppStart(func() {
		str += " World"
	}, 2)

	OnAppStart(func() {
		str += "Hello"
	}, 1)

	startFakeBookingApp()

	a.Equal("Hello World", str, "Failed to order OnAppStart")
}

// Ensure on app stop runs in order.
func TestOnAppStop(t *testing.T) {
	a := assert.New(t)
	startFakeBookingApp()
	i := ""
	OnAppStop(func() {
		i += "cruel world"
		t.Logf("i: %v \n", i)
	}, 2)
	OnAppStop(func() {
		i += "goodbye "
		t.Logf("i: %v \n", i)
	}, 1)
	go func() {
		time.Sleep(2 * time.Second)
		RaiseEvent(ENGINE_SHUTDOWN_REQUEST, nil)
	}()
	Run(0)
	a.Equal("goodbye cruel world", i, "Did not get shutdown events")
}

var (
	showRequest, _      = http.NewRequest("GET", "/hotels/3", nil)
	staticRequest, _    = http.NewRequest("GET", "/public/js/sessvars.js", nil)
	jsonRequest, _      = http.NewRequest("GET", "/hotels/3/booking", nil)
	plaintextRequest, _ = http.NewRequest("GET", "/hotels", nil)
)
