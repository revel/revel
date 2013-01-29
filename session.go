package revel

import (
	"net/http"
	"net/url"
	"strings"
)

// A signed cookie (and thus limited to 4kb in size).
// Restriction: Keys may not have a colon in them.
type Session map[string]string

type SessionPlugin struct{ EmptyPlugin }

func (p SessionPlugin) BeforeRequest(c *Controller) {
	c.Session = restoreSession(c.Request.Request)
}

func (p SessionPlugin) AfterRequest(c *Controller) {
	// Store the session (and sign it).
	var sessionValue string
	for key, value := range c.Session {
		sessionValue += "\x00" + key + ":" + value + "\x00"
	}
	sessionData := url.QueryEscape(sessionValue)
	c.SetCookie(&http.Cookie{
		Name:  CookiePrefix + "_SESSION",
		Value: Sign(sessionData) + "-" + sessionData,
		Path:  "/",
	})
}

func restoreSession(req *http.Request) Session {
	session := make(map[string]string)
	cookie, err := req.Cookie(CookiePrefix + "_SESSION")
	if err != nil {
		return Session(session)
	}

	// Separate the data from the signature.
	hyphen := strings.Index(cookie.Value, "-")
	if hyphen == -1 || hyphen >= len(cookie.Value)-1 {
		return Session(session)
	}
	sig, data := cookie.Value[:hyphen], cookie.Value[hyphen+1:]

	// Verify the signature.
	if Sign(data) != sig {
		INFO.Println("Session cookie signature failed")
		return Session(session)
	}

	ParseKeyValueCookie(data, func(key, val string) {
		session[key] = val
	})

	return Session(session)
}
