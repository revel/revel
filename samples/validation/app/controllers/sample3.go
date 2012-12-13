package controllers

import (
	"github.com/robfig/revel"
	"github.com/robfig/revel/samples/validation/app/models"
)

type Sample3 struct {
	*rev.Controller
}

func (c Sample3) Index() rev.Result {
	return c.Render()
}

func (c Sample3) HandleSubmit(user *models.User) rev.Result {
	user.Validate(c.Validation)

	// Handle errors
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Sample3.Index)
	}

	// Ok, display the created user
	return c.Render(user)
}
