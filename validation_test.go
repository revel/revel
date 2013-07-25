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

func validationTester(req *Request, fn func(c *Controller)) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	c := NewController(req, NewResponse(recorder))
	ValidationFilter(c, []Filter{func(c *Controller, _ []Filter) {
		fn(c)
	}})
	return recorder
}

// Test that errors are encoded into the _ERRORS cookie.
func TestValidationWithError(t *testing.T) {
	recorder := validationTester(buildEmptyRequest(), func(c *Controller) {
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
	recorder := validationTester(buildEmptyRequest(), func(c *Controller) {
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
	req := buildRequestWithCookie("REVEL_ERRORS", "invalid")
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
