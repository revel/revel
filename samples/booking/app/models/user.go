package models

import (
	"fmt"
	"play"
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

func (u *User) Validate(v *play.Validation) {
	v.Required(u.Username).Key("user.Username")
	v.MaxSize(u.Username, 15).Key("user.Username")
	v.MinSize(u.Username, 4).Key("user.Username")
	v.Match(u.Username, userRegex).Key("user.Username")

	v.Required(u.Password).Key("user.Password")
	v.MaxSize(u.Password, 15).Key("user.Password")
	v.MinSize(u.Password, 5).Key("user.Password")

	v.Required(u.Name).Key("user.Name")
	v.MaxSize(u.Name, 100).Key("user.Name")
}
