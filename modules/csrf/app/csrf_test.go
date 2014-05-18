package csrf

import (
	"bytes"
	"github.com/revel/revel"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

type fooController struct {
	*revel.Controller
}

func TestSecretInSession(t *testing.T) {
	resp := httptest.NewRecorder()
	getRequest, _ := http.NewRequest("GET", "http://www.example.com/", nil)
	c := revel.NewController(revel.NewRequest(getRequest), revel.NewResponse(resp))
	c.Session = make(revel.Session)
	filters := []revel.Filter{
		CsrfFilter,
		func(c *revel.Controller, fc []revel.Filter) {
			c.RenderHtml("{{ csrftoken . }}")
		},
	}

	filters[0](c, filters)

	if _, ok := c.Session["csrfSecret"]; !ok {
		t.Fatal("secret should be present in session")
	}
}

func TestPostWithoutToken(t *testing.T) {
	resp := httptest.NewRecorder()
	postRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)
	c := revel.NewController(revel.NewRequest(postRequest), revel.NewResponse(resp))
	c.Session = make(revel.Session)

	filters := []revel.Filter{
		CsrfFilter,
		func(c *revel.Controller, fc []revel.Filter) {
			c.RenderHtml("{{ csrftoken . }}")
		},
	}

	filters[0](c, filters)

	if resp.Code != 403 {
		t.Fatal("post without token should be forbidden")
	}
}

func TestNoReferrer(t *testing.T) {
	resp := httptest.NewRecorder()
	postRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)
	c := revel.NewController(revel.NewRequest(postRequest), revel.NewResponse(resp))

	c.Session = make(revel.Session)
	secret, _ := NewSecret()

	c.Session["csrfSecret"] = secret
	token := NewToken(c)

	// make a new request with the token
	data := url.Values{}
	data.Set("csrftoken", token)
	formPostRequest, _ := http.NewRequest("POST", "http://www.example.com/", bytes.NewBufferString(data.Encode()))
	formPostRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	formPostRequest.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	// and replace the old request
	c.Request = revel.NewRequest(formPostRequest)

	filters := []revel.Filter{
		CsrfFilter,
		func(c *revel.Controller, fc []revel.Filter) {
			c.RenderHtml("{{ csrftoken . }}")
		},
	}
	filters[0](c, filters)

	if resp.Code != 403 {
		t.Fatal("post without referer should be forbidden")
	}
}

func TestRefererHttps(t *testing.T) {
	resp := httptest.NewRecorder()
	postRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)
	c := revel.NewController(revel.NewRequest(postRequest), revel.NewResponse(resp))

	c.Session = make(revel.Session)
	secret, _ := NewSecret()

	c.Session["csrfSecret"] = secret
	token := NewToken(c)

	// make a new request with the token
	data := url.Values{}
	data.Set("csrftoken", token)
	formPostRequest, _ := http.NewRequest("POST", "https://www.example.com/", bytes.NewBufferString(data.Encode()))
	formPostRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	formPostRequest.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	formPostRequest.Header.Add("Referer", "http://www.example.com/")

	// and replace the old request
	c.Request = revel.NewRequest(formPostRequest)

	filters := []revel.Filter{
		CsrfFilter,
		func(c *revel.Controller, fc []revel.Filter) {
			c.RenderHtml("{{ csrftoken . }}")
		},
	}
	filters[0](c, filters)

	if resp.Code != 403 {
		t.Fatal("posts to https should have an https referer")
	}
}

func TestHeaderWithToken(t *testing.T) {
	resp := httptest.NewRecorder()
	postRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)
	c := revel.NewController(revel.NewRequest(postRequest), revel.NewResponse(resp))

	c.Session = make(revel.Session)
	secret, _ := NewSecret()

	c.Session["csrfSecret"] = secret
	token := NewToken(c)

	// make a new request with the token
	formPostRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)
	formPostRequest.Header.Add("X-CSRFToken", token)
	formPostRequest.Header.Add("Referer", "http://www.example.com/")

	// and replace the old request
	c.Request = revel.NewRequest(formPostRequest)

	filters := []revel.Filter{
		CsrfFilter,
		func(c *revel.Controller, fc []revel.Filter) {
			c.RenderHtml("{{ csrftoken . }}")
		},
	}

	filters[0](c, filters)

	if resp.Code == 403 {
		t.Fatal("post with http header token should be allowed")
	}
}

func TestFormPostWithToken(t *testing.T) {
	resp := httptest.NewRecorder()
	postRequest, _ := http.NewRequest("POST", "http://www.example.com/", nil)
	c := revel.NewController(revel.NewRequest(postRequest), revel.NewResponse(resp))

	c.Session = make(revel.Session)
	secret, _ := NewSecret()

	c.Session["csrfSecret"] = secret
	token := NewToken(c)

	// make a new request with the token
	data := url.Values{}
	data.Set("csrftoken", token)
	formPostRequest, _ := http.NewRequest("POST", "http://www.example.com/", bytes.NewBufferString(data.Encode()))
	formPostRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	formPostRequest.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	formPostRequest.Header.Add("Referer", "http://www.example.com/")

	// and replace the old request
	c.Request = revel.NewRequest(formPostRequest)

	filters := []revel.Filter{
		CsrfFilter,
		func(c *revel.Controller, fc []revel.Filter) {
			c.RenderHtml("{{ csrftoken . }}")
		},
	}

	filters[0](c, filters)

	if resp.Code == 403 {
		t.Fatal("form post with token should be allowed")
	}
}
