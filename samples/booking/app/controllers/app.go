package controllers

import (
	"play"
	"play/samples/booking/app/models"
)

func addUser(c *play.Controller) play.Result {
	if user := connected(c); user != nil {
		c.RenderArgs["user"] = user
	}
	return nil
}

func connected(c *play.Controller) *models.User {
	if c.RenderArgs["user"] != nil {
		return c.RenderArgs["user"].(*models.User)
	}
	if _, ok := c.Session["user"]; ok {
		// TODO: Return the user by username
	}
	return nil
}

type Application struct {
	*play.Controller
}

func (c Application) Index() play.Result {
	if connected(c.Controller) != nil {
		return c.Redirect(Hotels.Index)
	}
	title := "Home"
	return c.Render(title)
}

func (c Application) Register() play.Result {
	title := "Register"
	return c.Render(title)
}

func (c Application) SaveUser(user models.User, verifyPassword string) play.Result {
	c.Validation.Required(verifyPassword).Key("verifyPassword")
	c.Validation.Required(verifyPassword == user.Password).Key("verifyPassword").
		Message("Password does not match")
	user.Validate(c.Validation)

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Application.Register)
	}

	// TODO: Create user.
	c.Session["user"] = user.Username
	c.Flash.Success("Welcome, " + user.Name)
	return c.Redirect(Hotels.Index)
}

func init() {
	play.Intercept(addUser, play.BEFORE, &Application{})
}
