package controllers

import (
	"github.com/robfig/revel"
	"github.com/slogsdon/acvte/modules/auth"
)

type Session struct {
	*revel.Controller
}

func (c Session) init() {}

func (c Session) Index() revel.Result {
	return c.Redirect(Session.Create)
}

func (c Session) Create(username string, password string) revel.Result {
	if c.Request.Method == "POST" {
		user := auth.GetHash(username)

		if err := auth.RegisterSession(c.Controller, user.Password, password); err != nil {
			return c.Redirect(Session.Create)
		} else {
			return c.Redirect("/admin")
		}
	}
	return c.Render()
}

func (c Session) Destroy() revel.Result {
	return c.Render()
}
