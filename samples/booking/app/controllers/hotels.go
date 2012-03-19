package controllers

import (
	"play"
)

func checkUser(c *play.Controller) {
}

type Hotels struct {
	*play.Controller
}

func (c Hotels) Index() play.Result {
	return c.Render()
}
