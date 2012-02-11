package controllers

import (
	"net/http"
	"time"
	"play"
)

type Login struct {
	*play.Controller
}

func (c *Login) ShowLogin() play.Result {
	return c.Render()
}

func (c *Login) DoLogin(username, password string) play.Result {
	// Validate parameters.
	c.Validation.Required(username).
		Message("Please enter a username.")
	c.Validation.Required(password).
		Message("Please enter a password")
	c.Validation.Required(len(password) > 6).
		Message("Password must be at least 6 chars")

	// If validation failed, redirect back to the login form.
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect((*Login).ShowLogin)
	}

	// Check the credentials.
	if username != "user" || password != "password" {
		c.Flash.Error("Username or password not recognized")
		c.FlashParams()
		return c.Redirect((*Login).ShowLogin)
	}

	// Success.  Set the login cookie.
	c.SetCookie(&http.Cookie{
		Name: "Login",
		Value: "Success",
		Path: "/",
		Expires: time.Now().AddDate(0, 0, 7),
	})
	c.Flash.Success("Login successful.")
	return c.Redirect((*Application).Index)
}
