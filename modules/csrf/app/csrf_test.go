package csrf

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/revel/revel"
)

var testFilters = []revel.Filter{
	CsrfFilter,
	func(c *revel.Controller, fc []revel.Filter) {
		c.RenderHtml("{{ csrftoken . }}")
	},
}

func TestTokenInSession(t *testing.T) {
	resp := httptest.NewRecorder()
	getRequest, _ := http.NewRequest("GET", "http://www.example.com/", nil)
	c := revel.NewController(revel.NewRequest(getRequest), revel.NewResponse(resp))
	c.Session = make(revel.Session)

	testFilters[0](c, testFilters)

	if _, ok := c.Session["csrf_token"]; !ok {
		t.Fatal("token should be present in session")
	}
}

func TestPostWithoutToken(t *testing.T) {
	resp := httptest.NewRecorder()
	postRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)
	c := revel.NewController(revel.NewRequest(postRequest), revel.NewResponse(resp))
	c.Session = make(revel.Session)

	testFilters[0](c, testFilters)

	if c.Response.Status != 403 {
		t.Fatal("post without token should be forbidden")
	}
}

func TestNoReferrer(t *testing.T) {
	resp := httptest.NewRecorder()
	postRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)

	c := revel.NewController(revel.NewRequest(postRequest), revel.NewResponse(resp))
	c.Session = make(revel.Session)

	RefreshToken(c)
	token := c.Session["csrf_token"]

	// make a new request with the token
	data := url.Values{}
	data.Set("csrftoken", token)
	formPostRequest, _ := http.NewRequest("POST", "http://www.example.com/", bytes.NewBufferString(data.Encode()))
	formPostRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	formPostRequest.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	// and replace the old request
	c.Request = revel.NewRequest(formPostRequest)

	testFilters[0](c, testFilters)

	if c.Response.Status != 403 {
		t.Fatal("post without referer should be forbidden")
	}
}

func TestRefererHttps(t *testing.T) {
	resp := httptest.NewRecorder()
	postRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)
	c := revel.NewController(revel.NewRequest(postRequest), revel.NewResponse(resp))

	c.Session = make(revel.Session)

	RefreshToken(c)
	token := c.Session["csrf_token"]

	// make a new request with the token
	data := url.Values{}
	data.Set("csrftoken", token)
	formPostRequest, _ := http.NewRequest("POST", "https://www.example.com/", bytes.NewBufferString(data.Encode()))
	formPostRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	formPostRequest.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	formPostRequest.Header.Add("Referer", "http://www.example.com/")

	// and replace the old request
	c.Request = revel.NewRequest(formPostRequest)

	testFilters[0](c, testFilters)

	if c.Response.Status != 403 {
		t.Fatal("posts to https should have an https referer")
	}
}

func TestHeaderWithToken(t *testing.T) {
	resp := httptest.NewRecorder()
	postRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)
	c := revel.NewController(revel.NewRequest(postRequest), revel.NewResponse(resp))

	c.Session = make(revel.Session)

	RefreshToken(c)
	token := c.Session["csrf_token"]

	// make a new request with the token
	formPostRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)
	formPostRequest.Header.Add("X-CSRFToken", token)
	formPostRequest.Header.Add("Referer", "http://www.example.com/")

	// and replace the old request
	c.Request = revel.NewRequest(formPostRequest)

	testFilters[0](c, testFilters)

	if c.Response.Status == 403 {
		t.Fatal("post with http header token should be allowed")
	}
}

func TestFormPostWithToken(t *testing.T) {
	resp := httptest.NewRecorder()
	postRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)
	c := revel.NewController(revel.NewRequest(postRequest), revel.NewResponse(resp))

	c.Session = make(revel.Session)

	RefreshToken(c)
	token := c.Session["csrf_token"]

	// make a new request with the token
	data := url.Values{}
	data.Set("csrftoken", token)
	formPostRequest, _ := http.NewRequest("POST", "http://www.example.com/", bytes.NewBufferString(data.Encode()))
	formPostRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	formPostRequest.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	formPostRequest.Header.Add("Referer", "http://www.example.com/")

	// and replace the old request
	c.Request = revel.NewRequest(formPostRequest)

	testFilters[0](c, testFilters)

	if c.Response.Status == 403 {
		t.Fatal("form post with token should be allowed")
	}
}

func TestNoTokenInArgsWhenCORs(t *testing.T) {
	resp := httptest.NewRecorder()

	getRequest, _ := http.NewRequest("GET", "http://www.example1.com/", nil)
	getRequest.Header.Add("Referer", "http://www.example2.com/")

	c := revel.NewController(revel.NewRequest(getRequest), revel.NewResponse(resp))
	c.Session = make(revel.Session)

	testFilters[0](c, testFilters)

	if _, ok := c.RenderArgs["_csrftoken"]; ok {
		t.Fatal("RenderArgs should not contain token when not same origin")
	}
}
