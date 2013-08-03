package revel

import (
	"fmt"
	"github.com/streadway/simpleuuid"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// A signed cookie (and thus limited to 4kb in size).
// Restriction: Keys may not have a colon in them.
type Session map[string]string

const (
	SESSION_ID_KEY = "_ID"
	TS_KEY         = "_TS"
)

var expireAfterDuration time.Duration

func init() {
	// Set expireAfterDuration, default to 30 days if no value in config
	OnAppStart(func() {
		var err error
		if expiresString, ok := Config.String("session.expires"); !ok {
			expireAfterDuration = 30 * 24 * time.Hour
		} else if expireAfterDuration, err = time.ParseDuration(expiresString); err != nil {
			panic(fmt.Errorf("session.expires invalid: %s", err))
		}
	})
}

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

// Return a time.Time with session expiration date
func getSessionExpiration() time.Time {
	return time.Now().Add(expireAfterDuration)
}

// Returns an http.Cookie containing the signed session.
func (s Session) cookie() *http.Cookie {
	var sessionValue string
	ts := getSessionExpiration()
	s[TS_KEY] = getSessionExpirationCookie(ts)
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
		Name:    CookiePrefix + "_SESSION",
		Value:   Sign(sessionData) + "-" + sessionData,
		Path:    "/",
		Expires: ts.UTC(),
	}
}

func sessionTimeoutExpiredOrMissing(session Session) bool {
	if exp, present := session[TS_KEY]; !present {
		return true
	} else if expInt, _ := strconv.Atoi(exp); int64(expInt) < time.Now().Unix() {
		return true
	}
	return false
}

// Returns a Session pulled from signed cookie.
func getSessionFromCookie(cookie *http.Cookie) Session {
	session := make(Session)

	// Separate the data from the signature.
	hyphen := strings.Index(cookie.Value, "-")
	if hyphen == -1 || hyphen >= len(cookie.Value)-1 {
		return session
	}
	sig, data := cookie.Value[:hyphen], cookie.Value[hyphen+1:]

	// Verify the signature.
	if !Verify(data, sig) {
		INFO.Println("Session cookie signature failed")
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

func SessionFilter(c *Controller, fc []Filter) {
	c.Session = restoreSession(c.Request.Request)

	fc[0](c, fc[1:])

	// Store the session (and sign it).
	c.SetCookie(c.Session.cookie())
}

func restoreSession(req *http.Request) Session {
	session := make(Session)
	cookie, err := req.Cookie(CookiePrefix + "_SESSION")
	if err != nil {
		return session
	}

	return getSessionFromCookie(cookie)
}

func getSessionExpirationCookie(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}
