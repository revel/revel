package controllers

import (
	"github.com/robfig/revel"
	"github.com/robfig/revel/samples/chat/app/chatroom"
)

type Refresh struct {
	*rev.Controller
}

func (c Refresh) Index(user string) rev.Result {
	chatroom.Join(user)
	return c.Room(user)
}

func (c Refresh) Room(user string) rev.Result {
	subscription := chatroom.Subscribe()
	defer subscription.Cancel()
	events := subscription.Archive
	for i, _ := range events {
		if events[i].User == user {
			events[i].User = "you"
		}
	}
	return c.Render(user, events)
}

func (c Refresh) Say(user, message string) rev.Result {
	chatroom.Say(user, message)
	return c.Room(user)
}

func (c Refresh) Leave(user string) rev.Result {
	chatroom.Leave(user)
	return c.Redirect(Application.Index)
}
