package models

import "github.com/robfig/revel"

type User struct {
	Username        string
	FirstName       string
	LastName        string
	Age             int
	Password        string
	PasswordConfirm string
	Email           string
	EmailConfirm    string
	TermsOfUse      bool
}

func (user *User) Validate(v *revel.Validation) {
	v.Required(user.Username)
	v.MinSize(user.Username, 6)
	v.Required(user.FirstName)
	v.Required(user.LastName)
	v.Required(user.Age)
	v.Range(user.Age, 16, 120)
	v.Required(user.Password)
	v.MinSize(user.Password, 6)
	v.Required(user.PasswordConfirm)
	v.Required(user.PasswordConfirm == user.Password).
		Message("The passwords do not match.")
	v.Required(user.Email)
	v.Email(user.Email)
	v.Required(user.EmailConfirm)
	v.Required(user.EmailConfirm == user.Email).
		Message("The email addresses do not match")
	v.Required(user.TermsOfUse)
}
