package revel

import (
	"fmt"
	"net/http"
	"net/url"
)

// Flash represents a cookie that gets overwritten on each request.
// It allows data to be stored across one page at a time.
// This is commonly used to implement success or error messages.
// e.g. the Post/Redirect/Get pattern: http://en.wikipedia.org/wiki/Post/Redirect/Get
type Flash struct {
	Data, Out map[string]string
}

func (f Flash) Error(msg string, args ...interface{}) {
	if len(args) == 0 {
		f.Out["error"] = msg
	} else {
		f.Out["error"] = fmt.Sprintf(msg, args...)
	}
}

func (f Flash) Success(msg string, args ...interface{}) {
	if len(args) == 0 {
		f.Out["success"] = msg
	} else {
		f.Out["success"] = fmt.Sprintf(msg, args...)
	}
}

func FlashFilter(c *Controller, fc []Filter) {
	c.Flash = restoreFlash(c.Request.Request)
	c.RenderArgs["flash"] = c.Flash.Data

	fc[0](c, fc[1:])

	// Store the flash.
	var flashValue string
	for key, value := range c.Flash.Out {
		flashValue += "\x00" + key + ":" + value + "\x00"
	}
	c.SetCookie(&http.Cookie{
		Name:  CookiePrefix + "_FLASH",
		Value: url.QueryEscape(flashValue),
		Path:  "/",
	})
}

// Restore flash from a request.
func restoreFlash(req *http.Request) Flash {
	flash := Flash{
		Data: make(map[string]string),
		Out:  make(map[string]string),
	}
	if cookie, err := req.Cookie(CookiePrefix + "_FLASH"); err == nil {
		ParseKeyValueCookie(cookie.Value, func(key, val string) {
			flash.Data[key] = val
		})
	}
	return flash
}
