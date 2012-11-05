package models

import "github.com/mrjones/oauth"

type User struct {
	Username     string
	RequestToken *oauth.RequestToken
	AccessToken  *oauth.AccessToken
}

func FindOrCreate(username string) *User {
	if user, ok := db[username]; ok {
		return user
	}
	user := &User{Username: username}
	db[username] = user
	return user
}

var db = make(map[string]*User)
