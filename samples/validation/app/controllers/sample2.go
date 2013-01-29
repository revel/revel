package controllers

import (
	"github.com/robfig/revel"
)

type Sample2 struct {
	*revel.Controller
}

func (c Sample2) Index() revel.Result {
	return c.Render()
}

func (c Sample2) HandleSubmit(
	username, firstname, lastname string,
	age int,
	password, passwordConfirm, email, emailConfirm string,
	termsOfUse bool) revel.Result {

	// Validation rules
	c.Validation.Required(username)
	c.Validation.MinSize(username, 6)
	c.Validation.Required(firstname)
	c.Validation.Required(lastname)
	c.Validation.Required(age)
	c.Validation.Range(age, 16, 120)
	c.Validation.Required(password)
	c.Validation.MinSize(password, 6)
	c.Validation.Required(passwordConfirm)
	c.Validation.Required(passwordConfirm == password).Message("Your passwords do not match.")
	c.Validation.Required(email)
	c.Validation.Email(email)
	c.Validation.Required(emailConfirm)
	c.Validation.Required(emailConfirm == email).Message("Your email addresses do not match.")
	c.Validation.Required(termsOfUse == true).Message("Please agree to the terms.")

	// Handle errors
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Sample2.Index)
	}

	// Ok, display the created user
	return c.Render(username, firstname, lastname, age, password, email)
}
