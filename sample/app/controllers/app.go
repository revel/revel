package controllers

import "play"

type Application struct {
	*play.Controller
}

func (c *Application) Index() (*play.Result) {
	return c.Render()
}

func (c *Application) ShowApp() (*play.Result) {
	return c.Render()
}

