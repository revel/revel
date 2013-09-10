package controllers

import (
	"github.com/robfig/revel"
	"github.com/slogsdon/revel/modules/auth"
)

type Session struct {
	*revel.Controller
}

func (c Session) init() {}

func (c Session) Index() revel.Result {
	return c.Redirect(Session.Create)
}

func (c Session) Create() revel.Result {
	return c.Render()
}

func (c Session) Register(username string, password string) revel.Result {
	user := auth.GetHash(username)

	if err := auth.RegisterSession(c.Controller, user.Password, password); err != nil {
		return c.Redirect(Session.Create)
	} else {
		return c.Redirect(auth.RedirectTo)
	}
}

func (c Session) Destroy() revel.Result {
	auth.InvalidateSession(c)
	return c.Render()
}
