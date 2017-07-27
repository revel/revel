// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"net/http"
	"testing"
	"time"
)

func TestSessionRestore(t *testing.T) {
	expireAfterDuration = 0
	originSession := make(Session)
	originSession["foo"] = "foo"
	originSession["bar"] = "bar"
	cookie := originSession.Cookie()
	if !cookie.Expires.IsZero() {
		t.Error("incorrect cookie expire", cookie.Expires)
	}

	restoredSession := GetSessionFromCookie(cookie)
	for k, v := range originSession {
		if restoredSession[k] != v {
			t.Errorf("session restore failed session[%s] != %s", k, v)
		}
	}
}

func TestSessionExpire(t *testing.T) {
	expireAfterDuration = time.Hour
	session := make(Session)
	session["user"] = "Tom"
	var cookie *http.Cookie
	for i := 0; i < 3; i++ {
		cookie = session.Cookie()
		time.Sleep(time.Second)
		session = GetSessionFromCookie(cookie)
	}
	expectExpire := time.Now().Add(expireAfterDuration)
	if cookie.Expires.Unix() < expectExpire.Add(-time.Second).Unix() {
		t.Error("expect expires", cookie.Expires, "after", expectExpire.Add(-time.Second))
	}
	if cookie.Expires.Unix() > expectExpire.Unix() {
		t.Error("expect expires", cookie.Expires, "before", expectExpire)
	}

	session.SetNoExpiration()
	for i := 0; i < 3; i++ {
		cookie = session.Cookie()
		session = GetSessionFromCookie(cookie)
	}
	cookie = session.Cookie()
	if !cookie.Expires.IsZero() {
		t.Error("expect cookie expires is zero")
	}

	session.SetDefaultExpiration()
	cookie = session.Cookie()
	expectExpire = time.Now().Add(expireAfterDuration)
	if cookie.Expires.Unix() < expectExpire.Add(-time.Second).Unix() {
		t.Error("expect expires", cookie.Expires, "after", expectExpire.Add(-time.Second))
	}
	if cookie.Expires.Unix() > expectExpire.Unix() {
		t.Error("expect expires", cookie.Expires, "before", expectExpire)
	}
}
