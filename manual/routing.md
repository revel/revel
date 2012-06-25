---
title: Routing
layout: manual
---

Routes are defined in a separate `routes` file, using the original Play! syntax.

The basic syntax is:

	(METHOD) (URL Pattern) (Controller.Action)

This example demonstrates all of the features:

	# conf/routes
	# This file defines all application routes (Higher priority routes first)
	GET    /login                 Application.Login      <b># A simple path</b>
	GET    /hotels/?              Hotels.Index           <b># Match /hotels and /hotels/ (optional trailing slash)</b>
	GET    /hotels/{id}           Hotels.Show            <b># Extract a URI argument (matching /[^/]+/)</b>
	POST   /hotels/{<[0-9]+>id}   Hotels.Save            <b># URI arg with custom regex</b>
	WS     /hotels/{id}/feed      Hotels.Feed            <b># WebSockets.</b>
	POST   /hotels/{id}/{action}  Hotels.{action}        <b># Automatically route some actions.</b>
	GET    /public/               staticDir:public       <b># Map /app/public resources under /public/...</b>
	*      /{controller}/{action} {controller}.{action}  <b># Catch all; Automatic URL generation</b>

Let's go through the lines one at a time.

## A simple path

	GET    /login                 Application.Login

The simplest route uses an exact match on method and path.  It invokes the Login
action on the Application controller.

## Optional trailing slash

	GET    /hotels/?              Hotels.Index

Question marks are treated as in regular expressions: they allow the path to
match with or without the preceeding character.  This route invokes Hotels.Index
for both `/hotels` and `/hotels`

## URL Parameters

	GET    /hotels/{id}           Hotels.Show

Segments of the path may be matched and extracted.  By default, `{id}` will
match anything except a slash (`[^/]+`).  In this case, `/hotels/123` and
`/hotels/abc` would both be matched by this route.

Extracted parameters are available in the `Controller.Params` map, as well as
via action method parameters.  For example:

	func (c Hotels) Show(id int) rev.Result {
		...
	}

or

	func (c Hotels) Show() rev.Result {
		var id string = c.Params.Get("id")
		...
	}

or

	func (c Hotels) Show() rev.Result {
		var id int = c.Params.Bind("id", reflect.TypeOf(0))
		...
	}

## URL Parameter with Custom Regex

	POST   /hotels/{<[0-9]+>id}   Hotels.Save

Routes may also specify a regular expression with their parameters to restrict
what they may match.  The regular expression goes between <brackets>, before the
name.

In the example, we restrict the Hotel ID to be numerical.

## Websockets

	WS     /hotels/{id}/feed      Hotels.Feed

Websockets are routed in the same way as other requests, using a method
identifier of **WS**.

The corresponding action would have this signature:

	func (c Hotels) Feed(ws *websocket.Conn, id int) rev.Result {
		...
	}

## Static Serving

	GET    /public/               staticDir:public

For serving directories of static assets, Revel provides the **staticDir:**
directive.  This route tells Revel to use
[http.ServeFile](http://www.golang.org/pkg/net/http/#ServeFile) to serve
requests with a path prefix of `/public/` the corresponding static file within
the `public` directory.  (Refer to [organization](organization.html) for the
directory layout)

## Auto Routing

	POST   /hotels/{id}/{action}  Hotels.{action}
	*      /{controller}/{action} {controller}.{action}

URL argument extraction can also be used to determine the invoked action.

The first example route would effect the following routes:

	/hotels/1/Show    => Hotels.Show
	/hotels/2/Details => Hotels.Details

Similarly, the second example may be used to access any action in the
application:

	/Application/Login => Application.Login
	/Users/List        => Users.List

Using auto-routing as a catch-all (e.g. last route in the file) is useful for
quickly hooking up actions to non-vanity URLs.
