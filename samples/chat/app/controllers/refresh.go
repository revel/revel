package controllers

import (
	"github.com/revel/revel"
	"github.com/revel/revel/samples/chat/app/chatroom"
	"github.com/revel/revel/samples/chat/app/routes"
)

type Refresh struct {
	*revel.Controller
}

func (c Refresh) Index(user string) revel.Result {
	chatroom.Join(user)
	return c.Redirect(routes.Refresh.Room(user))
}

func (c Refresh) Room(user string) revel.Result {
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

func (c Refresh) Say(user, message string) revel.Result {
	chatroom.Say(user, message)
	return c.Redirect(routes.Refresh.Room(user))
}

func (c Refresh) Leave(user string) revel.Result {
	chatroom.Leave(user)
	return c.Redirect(Application.Index)
}
