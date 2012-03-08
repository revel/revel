package controllers

import (
	"net/http"
	"play"
	"time"
)

type Login struct {
	*play.Controller
}

// This is an interceptor that checks for the login cookie.
// If not present, redirect the user to the login page.
func CheckLogin(c *play.Controller) play.Result {
	if loginCookie, err := c.Request.Cookie("Login"); err == nil {
		if loginCookie.Value == "Success" {
			return nil
		}
	}
	return c.Redirect((*Login).ShowLogin)
}

func (c *Login) ShowLogin() play.Result {
	return c.Render()
}

func (c *Login) DoLogin(username, password string) play.Result {
	// Validate parameters.
	c.Validation.Required(username).Message("Please enter a username.")
	c.Validation.Required(password).Message("Please enter a password.")
	c.Validation.Required(len(password) > 6).Message("Password must be at least 6 chars.")

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
		Name:    "Login",
		Value:   "Success",
		Path:    "/",
		Expires: time.Now().AddDate(0, 0, 7),
	})
	c.Flash.Success("Login successful.")
	return c.Redirect((*Application).Index)
}

// Clear the cookie and redirect to the Login page.
func (c *Login) Logout() play.Result {
	c.SetCookie(&http.Cookie{
		Name:   "Login",
		Value:  "",
		Path:   "/",
		MaxAge: 0, // MaxAge = 0: Delete the cookie
	})
	c.Flash.Success("You have been logged out.")
	return c.Redirect((*Login).ShowLogin)
}
