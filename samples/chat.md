---
title: Chat room
layout: samples
---

The Chat app demonstrates:

* Using channels to implement a chat room (a pub-sub model).
* Using Comet and Websockets

Here are the contents of the app:

	chat/app/
		chatroom	       # Chat room routines
			chatroom.go

		controllers
			app.go         # The welcome screen, allowing user to pick a technology
			refresh.go     # Handlers for the "Active Refresh" chat demo
			longpolling.go # Handlers for the "Long polling" ("Comet") chat demo
			websocket.go   # Handlers for the "Websocket" chat demo

		views
			...            # HTML and Javascript

[Browse the code on Github](https://github.com/robfig/revel/tree/master/samples/chat)

## The Chat Room

First, let's look at how the chat room is implemented, in
[**chatroom.go**](https://github.com/robfig/revel/tree/master/samples/chat/app/chatroom/chatroom.go).

The chat room runs as an independent go-routine, started on initialization:

<pre class="prettyprint lang-go">
func init() {
	go chatroom()
}
</pre>

The `chatroom()` function simply selects on three channels and executes the
requested action.

<pre class="prettyprint lang-go">{% capture chatroom %}{% raw %}
var (
	// Send a channel here to get room events back.  It will send the entire
	// archive initially, and then new messages as they come in.
	subscribe = make(chan (chan<- Subscription), 10)
	// Send a channel here to unsubscribe.
	unsubscribe = make(chan (<-chan Event), 10)
	// Send events here to publish them.
	publish = make(chan Event, 10)
)

func chatroom() {
	archive := list.New()
	subscribers := list.New()

	for {
		select {
		case ch := <-subscribe:
			// Add subscriber to list and send back subscriber channel + chat log.
		case event := <-publish:
			// Send event to all subscribers and add to chat log.
		case unsub := <-unsubscribe:
			// Remove subscriber from subscriber list.
		}
	}
}
{% endraw %}{% endcapture %}{{ chatroom|escape }}</pre>

Let's see how each of those is implemented.

### Subscribe

<pre class="prettyprint lang-go">{% capture subscribe %}{% raw %}
	case ch := <-subscribe:
		var events []Event
		for e := archive.Front(); e != nil; e = e.Next() {
			events = append(events, e.Value.(Event))
		}
		subscriber := make(chan Event, 10)
		subscribers.PushBack(subscriber)
		ch <- Subscription{events, subscriber}
{% endraw %}{% endcapture %}{{ subscribe|escape }}</pre>

A Subscription is created with two properties:

* The chat log (archive)
* A channel that the subscriber can listen on to get new messages.

The Subscription is then sent back over the channel that the subscriber
supplied.


### Publish

<pre class="prettyprint lang-go">{% capture publish %}{% raw %}
	case event := <-publish:
		for ch := subscribers.Front(); ch != nil; ch = ch.Next() {
			ch.Value.(chan Event) <- event
		}
		if archive.Len() >= archiveSize {
			archive.Remove(archive.Front())
		}
		archive.PushBack(event)
{% endraw %}{% endcapture %}{{ publish|escape }}</pre>

The published event is sent to the subscribers' channels one by one.  Then the
event is added to the archive, which is trimmed if necessary.

### Unsubscribe

<pre class="prettyprint lang-go">{% capture unsub %}{% raw %}
	case unsub := <-unsubscribe:
		for ch := subscribers.Front(); ch != nil; ch = ch.Next() {
			if ch.Value.(chan Event) == unsub {
				subscribers.Remove(ch)
			}
		}
{% endraw %}{% endcapture %}{{ unsub|escape }}</pre>

The subscriber channel is removed from the list.

## Handlers

Now that you know how the chat room works, we can look at how the handlers
expose that functionality using different techniques.

### Active Refresh

The Active Refresh chat room javascript refreshes the page every 5 seconds to
get any new messages:

<pre class="prettyprint lang-js">
  // Scroll the messages panel to the end
  var scrollDown = function() {
    $('#thread').scrollTo('max')
  }

  // Reload the whole messages panel
  var refresh = function() {
    $('#thread').load('/refresh/room?user={{.user}} #thread .message', function() {
      scrollDown()
    })
  }

  // Call refresh every 5 seconds
  setInterval(refresh, 5000)
</pre>

> [Refresh/Room.html](https://github.com/robfig/revel/tree/master/samples/chat/app/views/Refresh/Room.html)

This is the handler to serve that:

<pre class="prettyprint lang-go">
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
</pre>

> [refresh.go](https://github.com/robfig/revel/tree/master/samples/chat/app/controllers/refresh.go)

It subscribes to the chatroom and passes the archive to the template to be
rendered (after changing the user name to "you" as necessary).

Nothing much to see here.

### Long Polling (Comet)

The Long Polling chat room javascript makes an ajax request that the server
keeps open until a new message comes in.  The javascript provides a
`lastReceived` timestamp to tell the server the last message it knows about.

<pre class="prettyprint lang-js">
  var lastReceived = 0
  var waitMessages = '/longpolling/room/messages?lastReceived='
  var say = '/longpolling/room/messages?user={{.user}}'

  $('#send').click(function(e) {
    var message = $('#message').val()
    $('#message').val('')
    $.post(say, {message: message})
  });

  // Retrieve new messages
  var getMessages = function() {
    $.ajax({
      url: waitMessages + lastReceived,
      success: function(events) {
        $(events).each(function() {
          display(this)
          lastReceived = this.Timestamp
        })
        getMessages()
      },
      dataType: 'json'
    });
  }
  getMessages();
</pre>

> [LongPolling/Room.html](https://github.com/robfig/revel/tree/master/samples/chat/app/views/LongPolling/Room.html)

and here is the handler

<pre class="prettyprint lang-go">{% capture WaitMessages %}{% raw %}
func (c LongPolling) WaitMessages(lastReceived int) revel.Result {
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
{% endraw %}{% endcapture %}{{ WaitMessages|escape }}</pre>

> [longpolling.go](https://github.com/robfig/revel/tree/master/samples/chat/app/controllers/longpolling.go)

In this implementation, it can simply block on the subscription channel
(assuming it has already sent back everything in the archive).

### Websocket

The Websocket chat room javascript opens a websocket connection as soon as the
user has loaded the chat room page.

<pre class="prettyprint lang-js">
  // Create a socket
  var socket = new WebSocket('ws://127.0.0.1:9000/websocket/room/socket?user={{.user}}')

  // Message received on the socket
  socket.onmessage = function(event) {
    display(JSON.parse(event.data))
  }

  $('#send').click(function(e) {
    var message = $('#message').val()
    $('#message').val('')
    socket.send(message)
  });
</pre>

> [WebSocket/Room.html](https://github.com/robfig/revel/tree/master/samples/chat/app/views/WebSocket/Room.html#L51)

The first thing to do is to subscribe to new events, join the room, and send
down the archive.  Here is what that looks like:

<pre class="prettyprint lang-go">{% capture guy %}{% raw %}
func (c WebSocket) RoomSocket(user string, ws *websocket.Conn) revel.Result {
	// Join the room.
	subscription := chatroom.Subscribe()
	defer subscription.Cancel()

	chatroom.Join(user)
	defer chatroom.Leave(user)

	// Send down the archive.
	for _, event := range subscription.Archive {
		if websocket.JSON.Send(ws, &event) != nil {
			// They disconnected
			return nil
		}
	}
{% endraw %}{% endcapture %}{{ guy|escape }}
</pre>

> [websocket.go](https://github.com/robfig/revel/tree/master/samples/chat/app/controllers/websocket.go#L17)

Next, we have to listen for new events from the subscription.  However, the
websocket library only provides a blocking call to get a new frame.  To select
between them, we have to wrap it:

<pre class="prettyprint lang-go">{% capture WebSocket2 %}{% raw %}
	// In order to select between websocket messages and subscription events, we
	// need to stuff websocket events into a channel.
	newMessages := make(chan string)
	go func() {
		var msg string
		for {
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				close(newMessages)
				return
			}
			newMessages <- msg
		}
	}()
{% endraw %}{% endcapture %}{{ WebSocket2|escape }}</pre>

> [websocket.go](https://github.com/robfig/revel/tree/master/samples/chat/app/controllers/websocket.go#L33)

Now we can select for new websocket messages on the `newMessages` channel.

The last bit does exactly that -- it waits for a new message from the websocket
(if the user has said something) or from the subscription (someone else in the
chat room has said something) and propagates the message to the other.

<pre class="prettyprint lang-go">{% capture WebSocket3 %}{% raw %}
	// Now listen for new events from either the websocket or the chatroom.
	for {
		select {
		case event := <-subscription.New:
			if websocket.JSON.Send(ws, &event) != nil {
				// They disconnected.
				return nil
			}
		case msg, ok := <-newMessages:
			// If the channel is closed, they disconnected.
			if !ok {
				return nil
			}

			// Otherwise, say something.
			chatroom.Say(user, msg)
		}
	}
	return nil
}
{% endraw %}{% endcapture %}{{ WebSocket3|escape }}</pre>

> [websocket.go](https://github.com/robfig/revel/tree/master/samples/chat/app/controllers/websocket.go#L48)

If we detect the websocket channel has closed, then we just return nil.
