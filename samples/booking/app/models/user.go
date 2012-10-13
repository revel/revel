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

func (user *User) Validate(v *rev.Validation) {
	v.Check(user.Username,
		rev.Required{},
		rev.MaxSize{15},
		rev.MinSize{4},
		rev.Match{userRegex},
	)

	ValidatePassword(v, user.Password).
		Key("user.Password")

	v.Check(user.Name,
		rev.Required{},
		rev.MaxSize{100},
	)
}

func ValidatePassword(v *rev.Validation, password string) *rev.ValidationResult {
	return v.Check(password,
		rev.Required{},
		rev.MaxSize{15},
		rev.MinSize{5},
	)
}
