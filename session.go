// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Session a signed cookie (and thus limited to 4kb in size).
// Restriction: Keys may not have a colon in them.
type Session map[string]string

// Session constants
const (
	SessionIDKey = "_ID"
	TimestampKey = "_TS"

	sessionKeyName = "session"
)

// expireAfterDuration is the time to live, in seconds, of a session cookie.
// It may be specified in config as "session.expires". Values greater than 0
// set a persistent cookie with a time to live as specified, and the value 0
// sets a session cookie.
var expireAfterDuration time.Duration

func init() {
	// Set expireAfterDuration, default to 30 days if no value in config
	OnAppStart(func() {
		var err error
		if expiresString, ok := Config.String("session.expires"); !ok {
			expireAfterDuration = 30 * 24 * time.Hour
		} else if expiresString == sessionKeyName {
			expireAfterDuration = 0
		} else if expireAfterDuration, err = time.ParseDuration(expiresString); err != nil {
			panic(fmt.Errorf("session.expires invalid: %s", err))
		}
	})
}

// ID retrieves from the cookie or creates a time-based UUID identifying this
// session.
func (s Session) ID() string {
	if sessionIDStr, ok := s[SessionIDKey]; ok {
		return sessionIDStr
	}

	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}

	s[SessionIDKey] = hex.EncodeToString(buffer)
	return s[SessionIDKey]
}

// getExpiration return a time.Time with the session's expiration date.
// If previous session has set to "session", remain it
func (s Session) getExpiration() time.Time {
	if expireAfterDuration == 0 || s[TimestampKey] == sessionKeyName {
		// Expire after closing browser
		return time.Time{}
	}
	return time.Now().Add(expireAfterDuration)
}

// Cookie returns an http.Cookie containing the signed session.
func (s Session) Cookie() *http.Cookie {
	var sessionValue string
	ts := s.getExpiration()
	s[TimestampKey] = getSessionExpirationCookie(ts)
	for key, value := range s {
		if strings.ContainsAny(key, ":\x00") {
			panic("Session keys may not have colons or null bytes")
		}
		if strings.Contains(value, "\x00") {
			panic("Session values may not have null bytes")
		}
		sessionValue += "\x00" + key + ":" + value + "\x00"
	}

	sessionData := url.QueryEscape(sessionValue)
	return &http.Cookie{
		Name:     CookiePrefix + "_SESSION",
		Value:    Sign(sessionData) + "-" + sessionData,
		Domain:   CookieDomain,
		Path:     "/",
		HttpOnly: true,
		Secure:   CookieSecure,
		Expires:  ts.UTC(),
	}
}

// sessionTimeoutExpiredOrMissing returns a boolean of whether the session
// cookie is either not present or present but beyond its time to live; i.e.,
// whether there is not a valid session.
func sessionTimeoutExpiredOrMissing(session Session) bool {
	if exp, present := session[TimestampKey]; !present {
		return true
	} else if exp == sessionKeyName {
		return false
	} else if expInt, _ := strconv.Atoi(exp); int64(expInt) < time.Now().Unix() {
		return true
	}
	return false
}

// GetSessionFromCookie returns a Session struct pulled from the signed
// session cookie.
func GetSessionFromCookie(cookie *http.Cookie) Session {
	session := make(Session)

	// Separate the data from the signature.
	hyphen := strings.Index(cookie.Value, "-")
	if hyphen == -1 || hyphen >= len(cookie.Value)-1 {
		return session
	}
	sig, data := cookie.Value[:hyphen], cookie.Value[hyphen+1:]

	// Verify the signature.
	if !Verify(data, sig) {
		WARN.Println("Session cookie signature failed")
		return session
	}

	ParseKeyValueCookie(data, func(key, val string) {
		session[key] = val
	})

	if sessionTimeoutExpiredOrMissing(session) {
		session = make(Session)
	}

	return session
}

// SessionFilter is a Revel Filter that retrieves and sets the session cookie.
// Within Revel, it is available as a Session attribute on Controller instances.
// The name of the Session cookie is set as CookiePrefix + "_SESSION".
func SessionFilter(c *Controller, fc []Filter) {
	c.Session = restoreSession(c.Request.Request)
	sessionWasEmpty := len(c.Session) == 0

	// Make session vars available in templates as {{.session.xyz}}
	c.ViewArgs["session"] = c.Session

	fc[0](c, fc[1:])

	// Store the signed session if it could have changed.
	if len(c.Session) > 0 || !sessionWasEmpty {
		c.SetCookie(c.Session.Cookie())
	}
}

// restoreSession returns either the current session, retrieved from the
// session cookie, or a new session.
func restoreSession(req *http.Request) Session {
	cookie, err := req.Cookie(CookiePrefix + "_SESSION")
	if err != nil {
		return make(Session)
	}
	return GetSessionFromCookie(cookie)
}

// getSessionExpirationCookie retrieves the cookie's time to live as a
// string of either the number of seconds, for a persistent cookie, or
// "session".
func getSessionExpirationCookie(t time.Time) string {
	if t.IsZero() {
		return sessionKeyName
	}
	return strconv.FormatInt(t.Unix(), 10)
}

// SetNoExpiration sets session to expire when browser session ends
func (s Session) SetNoExpiration() {
	s[TimestampKey] = sessionKeyName
}

// SetDefaultExpiration sets session to expire after default duration
func (s Session) SetDefaultExpiration() {
	delete(s, TimestampKey)
}
