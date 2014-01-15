package auth

import (
	"code.google.com/p/go.crypto/bcrypt"
	"github.com/robfig/revel"
	"time"
)

const (
	SESSION_KEY = "BasicAuth"
)

var (
	GetUser           func(string) *User
	GetAllowedActions func(*User) []string
	RedirectTo        string
)

func Check(c *revel.Controller) revel.Result {
	var result revel.Result
	
	if result = CheckAuth(c); result == nil {
		result = CheckActions(c)
	}

	return result
}

// CheckAuth is called to check for a valid session.
func CheckAuth(c *revel.Controller) revel.Result {
	session := c.Session[SESSION_KEY]
	result := Verify(session, c.Session.Id())
	
	if !result {
		Invalidate(c)
		c.Flash.Error("Session invalid. Please login.")
		return c.Redirect("/session/create")
	} else {
		session.UpdatedAt = time.Now()
		c.Session[SESSION_KEY] = session
	}
	return nil
}

func CheckActions(c *revel.Controller) revel.Result {
	s := c.Session[SESSION_KEY]
	
	if s == nil {
		c.Flash.Error("Session invalid. Please login.")
		return c.Redirect("/session/create")
	}

	for _, a := range s.AllowedActions {
        if a == c.Action {
            return c.Render()
        }
    }

	c.Flash.Error("You don't have permission to access this resource.")
	// TODO: what to render?
    return c.Render()
}

// Registers a valid session if password matches hash
func Register(c *revel.Controller, hash string, password string) error {
	h := []byte(hash)
	p := []byte(password)
	if err := ComparePassword(h, p); err != nil {
		return err
	}
	Set(c)
	return nil
}

func Set(c *revel.Controller) {
	s := Auth{
		Id:        c.Session.Id(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	c.Session[SESSION_KEY] = s
}

func SaveAllowedActions(c *revel.Controller, user *User) {
	if s := c.Session[SESSION_KEY]; s != nil {
		s.AllowedActions = GetAllowedActions(user)
		c.Session[SESSION_KEY] = s
	}
}

func Invalidate(c *revel.Controller) {
	c.Session[SESSION_KEY] = nil
}

// Verify checks stored session id against stored value
func Verify(session Auth, sid string) bool {
	if session == nil {
		return false
	}
	return sid == session.Id
}

// ComparePassword acts as a helper function to bcrypt.CompareHashAndPassword
// and is used to verify a plain-text password against a hashed password.
func ComparePassword(hash, attempt string) Error {
	err := bcrypt.CompareHashAndPassword(hash, []byte(attempt))
	return err
}

type Auth struct {
	Id             string
	AllowedActions []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
