// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session_test

//import (
//	"net/http"
//	"testing"
//	"time"
//
//	"github.com/revel/revel/session"
//	"github.com/aws/aws-sdk-go/aws/session"
//)
//
//// For test purposes
//func (cse *session.SessionCookieEngine) TestDecodeCookie(cookie *http.Cookie) {
//	// Reset the session
//	cse.Session = make(session.Session)
//	cse.DecodeCookie(GoCookie(*cookie))
//}
//
//
//func TestSessionRestore(t *testing.T) {
//	expireAfterDuration = 0
//	cse := NewSessionCookieEngine()
//	originSession := cse.Session
//	originSession["foo"] = "foo"
//	originSession["bar"] = "bar"
//	cookie := cse.Cookie()
//	if !cookie.Expires.IsZero() {
//		t.Error("incorrect cookie expire", cookie.Expires)
//	}
//
//	cse.DecodeCookie(GoCookie(*cookie))
//	restoredSession := cse.Session
//	for k, v := range originSession {
//		if restoredSession[k] != v {
//			t.Errorf("session restore failed session[%s] != %s", k, v)
//		}
//	}
//}
//
//func TestSessionExpire(t *testing.T) {
//	expireAfterDuration = time.Hour
//	cse := NewSessionCookieEngine()
//	session := cse.Session
//	session["user"] = "Tom"
//	var cookie *http.Cookie
//	for i := 0; i < 3; i++ {
//		cookie = cse.Cookie()
//		time.Sleep(time.Second)
//		cse.DecodeCookie(GoCookie(*cookie))
//		session = cse.Session
//	}
//	expectExpire := time.Now().Add(expireAfterDuration)
//	if cookie.Expires.Unix() < expectExpire.Add(-time.Second).Unix() {
//		t.Error("expect expires", cookie.Expires, "after", expectExpire.Add(-time.Second))
//	}
//	if cookie.Expires.Unix() > expectExpire.Unix() {
//		t.Error("expect expires", cookie.Expires, "before", expectExpire)
//	}
//
//	for i := 0; i < 3; i++ {
//		cookie = cse.Cookie()
//		cse.DecodeCookie(GoCookie(*cookie))
//		session = cse.Session
//		session.SetNoExpiration()
//	}
//	cookie = cse.Cookie()
//	if !cookie.Expires.IsZero() {
//		t.Error("expect cookie expires is zero")
//	}
//
//	session.SetDefaultExpiration()
//	cookie = cse.Cookie()
//	expectExpire = time.Now().Add(expireAfterDuration)
//	if cookie.Expires.Unix() < expectExpire.Add(-time.Second).Unix() {
//		t.Error("expect expires", cookie.Expires, "after", expectExpire.Add(-time.Second))
//	}
//	if cookie.Expires.Unix() > expectExpire.Unix() {
//		t.Error("expect expires", cookie.Expires, "before", expectExpire)
//	}
//}
