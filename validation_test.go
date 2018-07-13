// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// getRecordedCookie returns the recorded cookie from a ResponseRecorder with
// the given name. It utilizes the cookie reader found in the standard library.
func getRecordedCookie(recorder *httptest.ResponseRecorder, name string) (*http.Cookie, error) {
	r := &http.Response{Header: recorder.HeaderMap}
	for _, cookie := range r.Cookies() {
		if cookie.Name == name {
			return cookie, nil
		}
	}
	return nil, http.ErrNoCookie
}

// r.Original.URL.String()
func validationTester(req *Request, fn func(c *Controller)) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	c := NewTestController(recorder, req.In.GetRaw().(*http.Request))
	c.Request = req

	ValidationFilter(c, []Filter{I18nFilter, func(c *Controller, _ []Filter) {
		fn(c)
	}})
	return recorder
}

// Test that errors are encoded into the _ERRORS cookie.
func TestValidationWithError(t *testing.T) {
	recorder := validationTester(buildEmptyRequest().Request, func(c *Controller) {
		c.Validation.Required("")
		if !c.Validation.HasErrors() {
			t.Fatal("errors should be present")
		}
		c.Validation.Keep()
	})

	if cookie, err := getRecordedCookie(recorder, "REVEL_ERRORS"); err != nil {
		t.Fatal(err)
	} else if cookie.MaxAge < 0 {
		t.Fatalf("cookie should not expire")
	}
}

// Test that no cookie is sent if errors are found, but Keep() is not called.
func TestValidationNoKeep(t *testing.T) {
	recorder := validationTester(buildEmptyRequest().Request, func(c *Controller) {
		c.Validation.Required("")
		if !c.Validation.HasErrors() {
			t.Fatal("errors should not be present")
		}
	})

	if _, err := getRecordedCookie(recorder, "REVEL_ERRORS"); err != http.ErrNoCookie {
		t.Fatal(err)
	}
}

// Test that a previously set _ERRORS cookie is deleted if no errors are found.
func TestValidationNoKeepCookiePreviouslySet(t *testing.T) {
	req := buildRequestWithCookie("REVEL_ERRORS", "invalid").Request
	recorder := validationTester(req, func(c *Controller) {
		c.Validation.Required("success")
		if c.Validation.HasErrors() {
			t.Fatal("errors should not be present")
		}
	})

	if cookie, err := getRecordedCookie(recorder, "REVEL_ERRORS"); err != nil {
		t.Fatal(err)
	} else if cookie.MaxAge >= 0 {
		t.Fatalf("cookie should be deleted")
	}
}

func TestValidateMessageKey(t *testing.T) {
	Init("prod", "github.com/revel/revel/testdata", "")
	loadMessages(testDataPath)

	// Assert that we have the expected number of languages
	if len(MessageLanguages()) != 2 {
		t.Fatalf("Expected messages to contain no more or less than 2 languages, instead there are %d languages", len(MessageLanguages()))
	}
	req := buildRequestWithAcceptLanguages("nl").Request

	validationTester(req, func(c *Controller) {
		c.Validation.Required("").MessageKey("greeting")
		if msg := c.Validation.Errors[0].Message; msg != "Hallo" {
			t.Errorf("Failed expected message Hallo got %s", msg)
		}

		if !c.Validation.HasErrors() {
			t.Fatal("errors should not be present")
		}
	})

}
