package revel

import (
	"github.com/streadway/simpleuuid"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// A signed cookie (and thus limited to 4kb in size).
// Restriction: Keys may not have a colon in them.
type Session map[string]string

const (
	SESSION_ID_KEY = "_ID"
)

// Return a UUID identifying this session.
func (s Session) Id() string {
	if uuidStr, ok := s[SESSION_ID_KEY]; ok {
		return uuidStr
	}

	uuid, err := simpleuuid.NewTime(time.Now())
	if err != nil {
		panic(err) // I don't think this can actually happen.
	}
	s[SESSION_ID_KEY] = uuid.String()
	return s[SESSION_ID_KEY]
}

type SessionPlugin struct{ EmptyPlugin }

func (p SessionPlugin) BeforeRequest(c *Controller) {
	c.Session = restoreSession(c.Request.Request)
}

func (p SessionPlugin) AfterRequest(c *Controller) {
	// Store the session (and sign it).
	var sessionValue string
	for key, value := range c.Session {
		if strings.Contains(key, ":") {
			panic("Session keys may not have colons")
		}
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
