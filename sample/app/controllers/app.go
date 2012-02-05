package controllers

import (
	"play"
)

type Application struct {
	*play.Controller
}

func (c *Application) Index() play.Result {
	return c.Render(nil)
}

func (c *Application) ShowApp(id int) play.Result {
	return c.Render(id)
}
