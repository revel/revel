// Copyright (c) 2012-2018 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session_test

import (
	"fmt"
	"github.com/revel/revel"
	"github.com/revel/revel/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

// test the commands
func TestSessionString(t *testing.T) {
	session.InitSession(revel.RevelLog)
	a := assert.New(t)
	s := session.NewSession()
	s.Set("happy", "day")
	a.Equal("day", s.GetDefault("happy", nil, ""), fmt.Sprintf("Session Data %#v\n", s))

}

func TestSessionStruct(t *testing.T) {
	session.InitSession(revel.RevelLog)
	a := assert.New(t)
	s := session.NewSession()
	setSharedDataTest(s)
	a.Equal("test", s.GetDefault("happy.a.aa", nil, ""), fmt.Sprintf("Session Data %#v\n", s))

	stringMap := s.Serialize()
	s1 := session.NewSession()
	s1.Load(stringMap)
	testSharedData(s, s1, t, a)

}

func setSharedDataTest(s session.Session) {
	data := struct {
		A struct {
			Aa string
		}
		B int
		C string
		D float32
	}{A: struct {
		Aa string
	}{Aa: "test"},
		B: 5,
		C: "test",
		D: -325.25}
	s.Set("happy", data)
}
func testSharedData(s, s1 session.Session, t *testing.T, a *assert.Assertions) {
	// Compress the session to a string
	t.Logf("Original session %#v\n", s)
	t.Logf("New built session %#v\n", s1)
	data,err := s1.Get("happy.a.aa")
	a.Nil(err,"Expected nil")
	a.Equal("test", data, fmt.Sprintf("Session Data %#v\n", s))
	t.Logf("After test session %#v\n", s1)

}
