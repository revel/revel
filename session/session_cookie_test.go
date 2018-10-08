// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session_test

import (
	"testing"

	"github.com/revel/revel"
	"github.com/revel/revel/session"
	"github.com/stretchr/testify/assert"
	"net/http"
	"time"
)

func TestCookieRestore(t *testing.T) {
	a := assert.New(t)
	session.InitSession(revel.RevelLog)

	cse := revel.NewSessionCookieEngine()
	originSession := session.NewSession()
	setSharedDataTest(originSession)
	originSession["foo"] = "foo"
	originSession["bar"] = "bar"
	cookie := cse.GetCookie(originSession)
	if !cookie.Expires.IsZero() {
		t.Error("incorrect cookie expire", cookie.Expires)
	}

	restoredSession := session.NewSession()
	cse.DecodeCookie(revel.GoCookie(*cookie), restoredSession)
	a.Equal("foo",restoredSession["foo"])
	a.Equal("bar",restoredSession["bar"])
	testSharedData(originSession, restoredSession, t, a)
}

func TestCookieSessionExpire(t *testing.T) {
	session.InitSession(revel.RevelLog)
	cse := revel.NewSessionCookieEngine()
	cse.ExpireAfterDuration = time.Hour
	session := session.NewSession()
	session["user"] = "Tom"
	var cookie *http.Cookie
	for i := 0; i < 3; i++ {
		cookie = cse.GetCookie(session)
		time.Sleep(time.Second)

		cse.DecodeCookie(revel.GoCookie(*cookie), session)
	}
	expectExpire := time.Now().Add(cse.ExpireAfterDuration)
	if cookie.Expires.Unix() < expectExpire.Add(-time.Second).Unix() {
		t.Error("expect expires", cookie.Expires, "after", expectExpire.Add(-time.Second))
	}
	if cookie.Expires.Unix() > expectExpire.Unix() {
		t.Error("expect expires", cookie.Expires, "before", expectExpire)
	}

	// Test that the expiration time is zero for a "browser" session
	session.SetNoExpiration()
	cookie = cse.GetCookie(session)
	if !cookie.Expires.IsZero() {
		t.Error("expect cookie expires is zero")
	}

	// Check the default session is set
	session.SetDefaultExpiration()
	cookie = cse.GetCookie(session)
	expectExpire = time.Now().Add(cse.ExpireAfterDuration)
	if cookie.Expires.Unix() < expectExpire.Add(-time.Second).Unix() {
		t.Error("expect expires", cookie.Expires, "after", expectExpire.Add(-time.Second))
	}
	if cookie.Expires.Unix() > expectExpire.Unix() {
		t.Error("expect expires", cookie.Expires, "before", expectExpire)
	}
}
