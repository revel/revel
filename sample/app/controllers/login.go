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
	return c.Render(nil)
}

// TODO: Call with empty parameter, if necessary
func (c *Login) DoLogin(username, password string) play.Result {
	play.LOG.Println("DoLogin(", username, ",", password, ")")
	// TODO: Database
	// TODO: Validation
	if username == "user" && password == "password" {
		// Success.  Set the login cookie.
		c.SetCookie(&http.Cookie{
			Name: "Login",
			Value: "Success",
			Path: "/",
			Expires: time.Now().AddDate(0, 0, 7),
		})
		c.Response.Status = 404
		return c.Redirect((*Application).Index)
	} else {
		// Fail
		c.Flash.Error("Username or password not recognized")
		c.Response.Status = 404
		return c.Redirect((*Login).ShowLogin)
	}
	return nil
}
