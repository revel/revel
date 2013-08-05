---
title: Websockets
layout: manual
---

Revel provides support for [Websockets](http://en.wikipedia.org/wiki/WebSocket).

To handle a Websocket connection:

1. Add a route using the `WS` method.
2. Add an action that accepts a `*websocket.Conn` parameter.

For example, add this your `routes` file:

	WS /app/feed Application.Feed

Then write an action like this:

{% raw %}
<pre class="prettyprint lang-go">
import "code.google.com/p/go.net/websocket"

func (c App) Feed(user string, ws *websocket.Conn) revel.Result {
	...
}
</pre>
{% endraw %}

