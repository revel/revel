package controllers

import (
	"play"
)

type Login struct {
	*play.Controller
}

func (c *Login) Login() (*play.Result) {
	return c.Render(nil)
}
