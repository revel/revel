---
title: Routing
layout: manual
---

Routes are defined in a separate `routes` file.

The basic syntax is:

	(METHOD) (URL Pattern) (Controller.Action)

This example demonstrates all of the features:

	# conf/routes
	# This file defines all application routes (Higher priority routes first)
	GET    /login                 App.Login              # A simple path
	GET    /hotels/               Hotels.Index           # Match /hotels and /hotels/ (optional trailing slash)
	GET    /hotels/:id            Hotels.Show            # Extract a URI argument
	WS     /hotels/:id/feed       Hotels.Feed            # WebSockets.
	POST   /hotels/:id/:action    Hotels.:action         # Automatically route some actions.
	GET    /public/*filepath      Static.Serve("public") # Map /app/public resources under /public/...
	*      /:controller/:action   :controller.:action    # Catch all; Automatic URL generation

Let's go through the lines one at a time.  At the end, we'll see how to
accomplish **reverse routing** -- generating the URL to invoke a particular action.

## A simple path

	GET    /login                 App.Login

The simplest route uses an exact match on method and path.  It invokes the Login
action on the App controller.

## Trailing slashes

	GET    /hotels/               Hotels.Index

This route invokes `Hotels.Index` for both `/hotels` and `/hotels/`. The
reverse route to `Hotels.Index` will include the trailing slash.

Trailing slashes should not be used to differentiate between actions. The
simple path `/login` **will** be matched by a request to `/login/`.

## URL Parameters

	GET    /hotels/:id            Hotels.Show

Segments of the path may be matched and extracted.  The `:id` variable will
match anything except a slash.  For example, `/hotels/123` and
`/hotels/abc` would both be matched by this route.

Extracted parameters are available in the `Controller.Params` map, as well as
via action method parameters.  For example:

	func (c Hotels) Show(id int) revel.Result {
		...
	}

or

	func (c Hotels) Show() revel.Result {
		var id string = c.Params.Get("id")
		...
	}

or

	func (c Hotels) Show() revel.Result {
		var id int
		c.Params.Bind(&id, "id")
		...
	}

## Star parameters

	GET    /public/*filepath            Static.Serve("public")

The router recognizes a second kind of wildcard. The starred parameter must be
the last element in the path, and it matches all following path elements.

For example, in this case it will match any path beginning with "/public/", and
its value will be exactly the path substring that follows that prefix.

## Websockets

	WS     /hotels/:id/feed       Hotels.Feed

Websockets are routed in the same way as other requests, using a method
identifier of **WS**.

The corresponding action would have this signature:

	func (c Hotels) Feed(ws *websocket.Conn, id int) revel.Result {
		...
	}

## Static Serving

	GET    /public/*filepath            Static.Serve("public")
	GET    /favicon.ico                 Static.Serve("public","img/favicon.png")
	
For the 2 parameters version of Static.Serve, blank spaces are not allowed between
**"** and **,** due to how encoding/csv works.

For serving directories of static assets, Revel provides the **static** module,
which contains a single
[Static](http://godoc.org/github.com/robfig/revel/modules/static/app/controllers)
controller.  Its Serve action takes two parameters:

* prefix (string) - A (relative or absolute) path to the asset root.
* filepath (string) - A relative path that specifies the requested file.

(Refer to [organization](organization.html) for the directory layout)

## Fixed parameters

As demonstrated in the Static Serving section, routes may specify one or more
parameters to the action.  For example:

	GET    /products/:id     ShowList("PRODUCT")
	GET    /menus/:id        ShowList("MENU")

The provided argument(s) are bound to a parameter name using their position.  In
this case, the list type string would be bound to the name of the first action
parameter.

This could be helpful in situations where:

* you have a couple similar actions
* you have actions that do the same thing, but operate in different modes
* you have actions that do the same thing, but operate on different data types

## Auto Routing

	POST   /hotels/:id/:action    Hotels.:action
	*      /:controller/:action   :controller.:action

URL argument extraction can also be used to determine the invoked action.
Matching to controllers and actions is **case insensitive**.

The first example route line would effect the following routes:

	/hotels/1/show    => Hotels.Show
	/hotels/2/details => Hotels.Details

Similarly, the second example may be used to access any action in the
application:

	/app/login         => App.Login
	/users/list        => Users.List

Since matching to controllers and actions is case insensitive, the following
routes would also work:

	/APP/LOGIN         => App.Login
	/Users/List        => Users.List

Using auto-routing as a catch-all (e.g. last route in the file) is useful for
quickly hooking up actions to non-vanity URLs, especially in conjunction with
the reverse router..

## Reverse Routing

It is good practice to use a reverse router to generate URLs for a couple reasons:

* Avoids misspellings
* The compiler ensures that reverse routes have the right number and type of
  parameters.
* Localizes URL changes to one place: the routes file.

Upon building your application, Revel generates an `app/routes` package.  Use it
with a statement of the form:

<pre class="prettyprint lang-go">
routes.Controller.Action(param1, param2)
</pre>

The above statement returns a URL (type string) to Controller.Action with the
given parameters.  Here is a more complete example:

<pre class="prettyprint lang-go">{% capture html %}
import (
	"github.com/robfig/revel"
	"project/app/routes"
)

type App struct { *revel.Controller }

// Show a form
func (c App) ViewForm(username string) revel.Result {
	return c.Render(username)
}

// Process the submitted form.
func (c App) ProcessForm(username, input string) revel.Result {
	...
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.Flash.Error("Form invalid. Try again.")
		return c.Redirect(routes.App.ViewForm(username))  // <--- REVERSE ROUTE
	}
	c.Flash.Success("Form processed!")
	return c.Redirect(routes.App.ViewConfirmation(username, input))  // <--- REVERSE ROUTE
}{% endcapture %}{{ html|escape }}
</pre>


<div class="alert alert-info"><strong>Limitation:</strong> Only primitive
parameters to a route are typed due to the possibility of circular imports.
Non-primitive parameters are typed as interface{}.
</div>
