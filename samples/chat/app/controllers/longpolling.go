package controllers

import (
	"github.com/robfig/revel"
	"github.com/robfig/revel/samples/chat/app/chatroom"
)

type LongPolling struct {
	*rev.Controller
}

func (c LongPolling) Room(user string) rev.Result {
	chatroom.Join(user)
	return c.Render(user)
}

func (c LongPolling) Say(user, message string) rev.Result {
	chatroom.Say(user, message)
	return nil
}

func (c LongPolling) WaitMessages(lastReceived int) rev.Result {
	subscription := chatroom.Subscribe()
	defer subscription.Cancel()

	// See if anything is new in the archive.
	var events []chatroom.Event
	for _, event := range subscription.Archive {
		if event.Timestamp > lastReceived {
			events = append(events, event)
		}
	}

	// If we found one, grand.
	if len(events) > 0 {
		return c.RenderJson(events)
	}

	// Else, wait for something new.
	event := <-subscription.New
	return c.RenderJson([]chatroom.Event{event})
}

func (c LongPolling) Leave(user string) rev.Result {
	chatroom.Leave(user)
	return c.Redirect(Application.Index)
}
