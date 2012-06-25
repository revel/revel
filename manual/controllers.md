---
title: Overview
layout: manual
---

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

// These provide a unified view of the request params.
// Includes:
// - URL query string
// - Form values
// - File uploads
type Params struct {
	url.Values
	Files map[string][]*multipart.FileHeader
}

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

