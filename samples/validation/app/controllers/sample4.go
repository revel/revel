package controllers

import (
	"github.com/robfig/revel"
	"github.com/robfig/revel/samples/validation/app/models"
)

type Sample4 struct {
	*rev.Controller
}

func (c Sample4) Index() rev.Result {
	return c.Render()
}

func (c Sample4) HandleSubmit(user *models.User) rev.Result {
	user.Validate(c.Validation)

	// Handle errors
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Sample4.Index)
	}

	// Ok, display the created user
	return c.Render(user)
}
