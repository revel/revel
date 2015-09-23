package revel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testAssetsFilters = []Filter{
	AssetsFilter,
	func(c *Controller, fc []Filter) {
		c.RenderHtml("foo")
	},
}

func TestAssetsFilter(t *testing.T) {
	startFakeBookingApp()
	resp := httptest.NewRecorder()
	getRequest, _ := http.NewRequest("GET", "/assets/stylesheets/app.css", nil)
	c := NewController(NewRequest(getRequest), NewResponse(resp))
	testAssetsFilters[0](c, testAssetsFilters)

	rightResponseBody := `a {  color: red; }`
	body := strings.Replace(resp.Body.String(), "\n", "", -1)

	if !strings.Contains(body, rightResponseBody) {
		t.Fatal("Assets response expect", rightResponseBody, " but: ", body)
	}

	if resp.Code != 200 {
		t.Fatal("Assets not work, response status: ", resp.Code)
	}
}
