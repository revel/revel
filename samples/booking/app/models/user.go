package models

import (
	"fmt"
	"github.com/robfig/revel"
	"regexp"
)

type User struct {
	UserId                   int
	Username, Password, Name string
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

	v.Check(u.Password,
		rev.Required{},
		rev.MaxSize{15},
		rev.MinSize{5},
	).Key("user.Password")

	v.Check(u.Name,
		rev.Required{},
		rev.MaxSize{100},
	).Key("user.Name")
}
