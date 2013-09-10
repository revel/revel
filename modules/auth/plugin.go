package auth

import (
	"code.google.com/p/go.crypto/bcrypt"
	"github.com/robfig/revel"
	"github.com/robfig/revel/cache"
	"reflect"
	"time"
)

const (
	SESSION_KEY = "BasicAuthSessionId"
)

var (
	GetHash    func(string) *User
	RedirectTo string
	SessionId  string
)

// CheckSession is called to check for a valid session.
func CheckSession(c *revel.Controller) revel.Result {
	result := false
	if value, ok := c.Session[SESSION_KEY]; ok {
		result = VerifySession(value)
	}
	if !result {
		c.Flash.Error("Session invalid. Please login.")
		return c.Redirect("/session/create")
	}
	return nil
}

// Reisters a valid session
func RegisterSession(c *revel.Controller, hash string, password string) error {
	h := []byte(hash)
	p := []byte(password)
	SessionId = c.Session.Id()
	if err := bcrypt.CompareHashAndPassword(h, p); err != nil {
		return err
	}
	SetSession(c)
	return nil
}

func SetSession(c *revel.Controller) {
	c.Session[SESSION_KEY] = c.Session.Id()
	s := Session{
		Id:        c.Session.Id(),
		Data:      "true",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	cache.Set(c.Session.Id()+SESSION_KEY, s, 30*time.Minute)
}

func InvalidateSession() {
	go cache.Delete(SessionId + SESSION_KEY)
}

// VerifySession checks stored session id against stored value
func VerifySession(sid string) bool {
	var session Session
	if err := cache.Get(SessionId+SESSION_KEY, &session); err != nil {
		return false
	}
	return sid == session.Id
}

type Session struct {
	Id        string
	Data      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}
