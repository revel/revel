---
title: Concepts
layout: manual
---

Revel makes it easy to build web applications using the Model-View-Controller
(MVC) pattern by relying on conventions that require a certain structure in your
application.  In return, it is very light on configuration and enables an
extremely fast development cycle.

## MVC

Here is a quick summary:

- *Models* are the essential data objects that describe your application domain.
   Models also contain domain-specific logic for querying and updating the data.
- *Views* describe how data is presented and manipulated. In our case, this is
   the template that is used to present data and controls to the user.
- *Controllers* handle the request execution.  They perform the user's desired
   action, they decide which View to display, and they prepare and provide the
   necessary data to the View for rendering.

There are many excellent overviews of MVC structure online.  In particular, the
one provided by [Play! Framework](http://www.playframework.org) matches our model exactly.

## Goroutine per Request

Revel builds on top of the Go HTTP server, which creates a go-routine
(lightweight thread) to process each incoming request.  The implication is that
your code is free to block, but it must handle concurrent request processing.

## Controllers and Actions

Each HTTP request invokes an **action**, which handles the request and writes the
response. Related **actions** are grouped into **controllers**.

***

A **Controller** is any type that embeds `rev.Controller` (directly or indirectly).

Typically:
<pre class="prettyprint lang-go">
type AppController struct {
  *rev.Controller
}
</pre>

(Currently `rev.Controller` must be embedded as the first type in your struct)

The `rev.Controller` is the context for the request.  It contains the request
and response data.  Please refer to [the godoc](../docs/godoc/mvc.html#Controller)
for the full story, but here is the definition (along with definitions of helper types):

<pre class="prettyprint lang-go">
type Controller struct {
	Name       string
	Type       *ControllerType
	MethodType *MethodType

	Request  *Request
	Response *Response

	Flash      Flash                  // User cookie, cleared after each request.
	Session    Session                // Session, stored in cookie, signed.
	Params     Params                 // Parameters from URL and form (including multipart).
	Args       map[string]interface{} // Per-request scratch space.
	RenderArgs map[string]interface{} // Args passed to the template.
	Validation *Validation            // Data validation helpers
	Txn        *sql.Tx                // Nil by default, but may be used by the app / plugins
}

// Flash represents a cookie that gets overwritten on each request.
// It allows data to be stored across one page at a time.
// This is commonly used to implement success or error messages.
// e.g. the Post/Redirect/Get pattern: http://en.wikipedia.org/wiki/Post/Redirect/Get
type Flash struct {
	Data, Out map[string]string
}

// These provide a unified view of the request params.
// Includes:
// - URL query string
// - Form values
// - File uploads
type Params struct {
	url.Values
	Files map[string][]*multipart.FileHeader
}

// A signed cookie (and thus limited to 4kb in size).
// Restriction: Keys may not have a colon in them.
type Session map[string]string

type Request struct {
	*http.Request
	ContentType string
}

type Response struct {
	Status      int
	ContentType string
	Headers     http.Header
	Cookies     []*http.Cookie

	Out http.ResponseWriter
}
</pre>

As part of handling a HTTP request, Revel instantiates an instance of your
Controller, and it sets all of these properties on the embedded
`rev.Controller`.  Therefore, Revel does not share Controller instances between
requests.

***

An **Action** is any method on a **Controller** that meets the following criteria:
* is exported
* returns a rev.Result

For example:

<pre class="prettyprint lang-go">
func (c AppController) ShowLogin(username string) rev.Result {
	..
	return c.Render(username)
}
</pre>

The example invokes rev.Controller.Render to execute a template, passing it the
username as a parameter.  There are many methods on **rev.Controller** that
produce **rev.Result**, but applications are also free to create their own.

## Results

A Result is anything conforming to the interface:
<pre class="prettyprint lang-go">
type Result interface {
	Apply(req *Request, resp *Response)
}
</pre>

Typically, nothing is written to the response until the **action** has returned
a Result.  At that point, Revel writes response headers and cookies
(e.g. setting the session cookie), and then invokes `Result.Apply` to write the
actual response content.

(The action may choose to write directly to the response, but this would be
expected only in exceptional cases.  In those cases, it would have to handle
saving the Session and Flash data itself, for example)
