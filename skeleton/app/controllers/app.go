package controllers

import (
	"github.com/robfig/revel"
	"time"
)

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
	year := time.Now().Year()
	return c.Render(year)
}
