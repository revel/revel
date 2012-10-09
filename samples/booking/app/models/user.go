package models

import (
	"fmt"
	"github.com/robfig/revel"
	"regexp"
)

type User struct {
	UserId                   int
	Username, Password, Name string
	HashedPassword []byte
}

func (u *User) String() string {
	return fmt.Sprintf("User(%s)", u.Username)
}

var userRegex = regexp.MustCompile("^\\w*$")

func (u *User) Validate(v *rev.Validation) {
	v.Check(u.Username,
		rev.Required{},
		rev.MaxSize{15},
		rev.MinSize{4},
		rev.Match{userRegex},
	).Key("user.Username")

	ValidatePassword(v, u.Password).Key("user.Password")

	v.Check(u.Name,
		rev.Required{},
		rev.MaxSize{100},
	).Key("user.Name")
}

func ValidatePassword(v *rev.Validation, password string) *rev.ValidationResult {
	return v.Check(password,
		rev.Required{},
		rev.MaxSize{15},
		rev.MinSize{5},
	)
}
