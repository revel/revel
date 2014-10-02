package controllers

import (
	"github.com/revel/revel"
)

type App struct {
	*revel.Controller
}

type FileInfo struct {
	ContentType string
	Filename    string
	RealFormat  string `json:",omitempty"`
	Resolution  string `json:",omitempty"`
	Size        int
	Status      string `json:",omitempty"`
}

func (c *App) Before() revel.Result {
	// Rendering useful info here.
	c.RenderArgs["action"] = c.Controller.Action

	return nil
}

func init() {
	revel.InterceptMethod((*App).Before, revel.BEFORE)
}
