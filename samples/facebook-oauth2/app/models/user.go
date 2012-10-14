package models

import "math/rand"

type User struct {
	Uid         int
	AccessToken string
}

var db = make(map[int]*User)

func GetUser(id int) *User {
	return db[id]
}

func NewUser() *User {
	user := &User{Uid: rand.Intn(10000)}
	db[user.Uid] = user
	return user
}
