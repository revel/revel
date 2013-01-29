package controllers

import (
	"github.com/robfig/revel"
)

type Sample1 struct {
	*revel.Controller
}

func (c Sample1) Index() revel.Result {
	return c.Render()
}

func (c Sample1) HandleSubmit(
	username, firstname, lastname string,
	age int,
	password, passwordConfirm, email, emailConfirm string,
	termsOfUse bool) revel.Result {

	// Validation rules
	c.Validation.Required(username).Message("Username is required.")
	c.Validation.MinSize(username, 6).Message("Username must be at least 6 characters.")
	c.Validation.Required(firstname).Message("First name is required.")
	c.Validation.Required(lastname).Message("Last name is required.")
	c.Validation.Required(age).Message("Age is required.")
	c.Validation.Range(age, 16, 120).Message("Age must be between 16 and 120.")
	c.Validation.Required(password).Message("Password is required.")
	c.Validation.MinSize(password, 6).Message("Password must be greater than 6 characters.")
	c.Validation.Required(passwordConfirm).Message("Please confirm your password.")
	c.Validation.Required(passwordConfirm == password).Message("Your passwords do not match.")
	c.Validation.Required(email).Message("Email is required.")
	c.Validation.Email(email).Message("A valid email is required.")
	c.Validation.Required(emailConfirm).Message("Please confirm your email address.")
	c.Validation.Required(emailConfirm == email).Message("Your email addresses do not match.")
	c.Validation.Required(termsOfUse == true).Message("Please agree to the terms.")

	// Handle errors
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Sample1.Index)
	}

	// Ok, display the created user
	return c.Render(username, firstname, lastname, age, password, email)
}
