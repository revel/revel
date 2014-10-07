package csrf

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/url"
)

import "github.com/revel/revel"

// allowMethods are HTTP methods that do NOT require a token
var allowedMethods = map[string]bool{
	"GET":     true,
	"HEAD":    true,
	"OPTIONS": true,
	"TRACE":   true,
}

func RandomString(length int) (string, error) {
	buffer := make([]byte, int(math.Ceil(float64(length)/2)))
	if _, err := io.ReadFull(rand.Reader, buffer); err != nil {
		return "", nil
	}
	str := hex.EncodeToString(buffer)
	return str[:length], nil
}

func RefreshToken(c *revel.Controller) {
	token, err := RandomString(64)
	if err != nil {
		panic(err)
	}
	c.Session["csrf_token"] = token
}

func CsrfFilter(c *revel.Controller, fc []revel.Filter) {
	token, foundToken := c.Session["csrf_token"]

	if !foundToken {
		RefreshToken(c)
	}

	// TODO: Add hook for csrf exempt

	referer, refErr := url.Parse(c.Request.Header.Get("Referer"))

	// If the Request method isn't in the white listed methods
	if !allowedMethods[c.Request.Method] {
		// Token wasn't present at all
		if !foundToken {
			c.Result = c.Forbidden("REVEL CSRF: Session token missing.")
			return
		}

		// Referrer header is invalid
		if refErr != nil {
			c.Result = c.Forbidden("REVEL CSRF: HTTP Referer malformed.")
			return
		}

		// Same origin
		if !sameOrigin(c.Request.URL, referrer) {
			c.Result = c.Forbidden("REVEL CSRF: Same origin mismatch.")
			return
		}

		var requestToken string
		// First check for token in post data
		if c.Request.Method == "POST" {
			requestToken = c.Request.FormValue("csrftoken")
		}

		// Then check for token in custom headers, as with AJAX
		if requestToken == "" {
			requestToken = c.Request.Header.Get("X-CSRFToken")
		}

		if requestToken == "" || !compareToken(requestToken, csrfSecret) {
			c.Result = c.Forbidden("REVEL CSRF: Invalid token.")
			return
		}
	}

	fc[0](c, fc[1:])

	// Only add token to RenderArgs if the request is: not AJAX, not missing referrer header, and is same origin.
	if c.Request.Header.Get("X-CSRFToken") == "" && refErr == nil && sameOrigin(c.Request.URL, referrer) {
		c.RenderArgs["_csrftoken"] = token
	}
}

func compareToken(requestToken, token string) bool {
	// ConstantTimeCompare will panic if the []byte aren't the same length
	if len(requestToken) != len(token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(requestToken), []byte(token)) == 1
}

// Validates same origin policy
func sameOrigin(u1, u2 *url.URL) (bool, error) {
	return u1.Scheme == u2.Scheme && u1.Host == u2.Host
}

func init() {
	revel.TemplateFuncs["csrftoken"] = func(renderArgs map[string]interface{}) template.HTML {
		tokenFunc, ok := renderArgs["_csrftoken"]
		if !ok {
			panic("REVEL CSRF: _csrftoken missing from RenderArgs.")
		}
		return template.HTML(fmt.Sprintf(`<input type="hidden" name="csrftoken" value="%s">`, tokenFunc.(func() string)()))
	}

	revel.TemplateFuncs["csrftokenraw"] = func(renderArgs map[string]interface{}) template.HTML {
		tokenFunc, ok := renderArgs["_csrftoken"]
		if !ok {
			panic("REVEL CSRF: _csrftoken missing from RenderArgs.")
		}
		return template.HTML(tokenFunc.(func() string)())
	}
}
