package csrf

import (
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"github.com/revel/revel"
	"github.com/streadway/simpleuuid"
	"html/template"
	"io"
	"net/url"
	"time"
)

var allowedMethods = map[string]bool{
	"GET":     true,
	"HEAD":    true,
	"OPTIONS": true,
	"TRACE":   true,
}

func NewToken(c *revel.Controller) string {
	token := c.Request.Header.Get("Csrf-Token")
	if token == "" {
		token = saltedToken(c.Session["csrfSecret"])
		c.Request.Header.Set("Csrf-Token", token)
	}
	return token
}

func NewSecret() (simpleuuid.UUID, error) {
	secret, err := simpleuuid.NewTime(time.Now())
	return secret, err
}

func RefreshSecret(c *revel.Controller) {
	csrfSecret, err := NewSecret()
	if err != nil {
		panic(err)
	}
	c.Session["csrfSecret"] = csrfSecret.String()
}

func CsrfFilter(c *revel.Controller, fc []revel.Filter) {
	csrfSecret, foundSecret := c.Session["csrfSecret"]

	if !foundSecret {
		RefreshSecret(c)
	}

	// TODO: Add a hook for csrf exempt?
	// Whatever the flag, it needs to be visible during the request cycle...

	// If the Request method isn't in the white listed methods
	if !allowedMethods[c.Request.Method] {
		// Token wasn't present at all
		if !foundSecret {
			reject(c)
			return
		}
		// Referrer header isn't present
		referer := c.Request.Referer()
		if referer == "" {
			reject(c)
			return
		}
		// Referer is invalid, or the requested
		// page is HTTPS but the Referer is NOT HTTPS
		refUrl, err := url.Parse(referer)
		if err != nil || c.Request.URL.Scheme == "https" && refUrl.Scheme != "https" {
			reject(c)
			return
		}

		var requestCsrfToken string
		// First check post data
		if c.Request.Method == "POST" {
			requestCsrfToken = c.Request.FormValue("csrftoken")
		}

		// Then check custom headers, as with AJAX
		if requestCsrfToken == "" {
			requestCsrfToken = c.Request.Header.Get("X-CSRFToken")
		}

		if requestCsrfToken == "" || !checkToken(requestCsrfToken, csrfSecret) {
			reject(c)
			return
		}
	}

	fc[0](c, fc[1:])

	c.RenderArgs["_csrftoken"] = func() string {
		return NewToken(c)
	}
}

func createToken(salt, secret string) string {
	hash := sha1.New()
	io.WriteString(hash, salt+secret)
	return salt + base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func checkToken(requestCsrfToken, secret string) bool {
	csrfToken := createToken(requestCsrfToken[0:10], secret)
	// ConstantTimeCompare will panic if the []byte aren't the same length
	if len(requestCsrfToken) != len(csrfToken) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(requestCsrfToken), []byte(csrfToken)) == 1
}

func generateSalt(length int) string {
	salt, err := simpleuuid.NewTime(time.Now())
	if err != nil {
		panic(err)
	}
	return salt.String()[0:10]
}

func saltedToken(secret string) string {
	return createToken(generateSalt(10), secret)
}

func reject(c *revel.Controller) {
	c.Response.Out.WriteHeader(403)
}

func init() {
	revel.TemplateFuncs["csrftoken"] = func(renderArgs map[string]interface{}) template.HTML {
		tokenFunc, ok := renderArgs["_csrftoken"]
		if !ok {
			panic("_csrftoken missing from RenderArgs.")
		}
		return template.HTML(fmt.Sprintf(`<input type="hidden" name="csrftoken" value="%s">`, tokenFunc.(func() string)()))
	}
}
