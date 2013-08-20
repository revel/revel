package controllers

import (
	"github.com/robfig/revel"
	"github.com/robfig/revel/modules/auth"
	// "code.google.com/p/go.crypto/bcrypt"
)

type Session struct {
	*revel.Controller
}

func (c Session) init() {}

func (c Session) Index() revel.Result {
	return c.Redirect("/session/create")
}

func (c Session) Create() revel.Result {
	return c.Render()
}

func (c Session) Register(username string, password string) revel.Result {
	panic(auth.SessionId)
	return c.Render()
}

func (c Session) Destroy() revel.Result {
	return c.Render()
}
