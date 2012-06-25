---
title: Results
layout: manual
---

Actions must return a [`rev.Result`](../docs/godoc/results.html#Result), which
handles the response generation.  It adheres to a simple interface:

<pre class="prettyprint lang-go">
type Result interface {
	Apply(req *Request, resp *Response)
}
</pre>

[`rev.Controller`](../docs/godoc/mvc.html#Controller) provides a number of
methods to produce Results:
* Render(...) - render a template
* Redirect(...) - redirect to another action or URL

**Note:** Actions that write to the response themselves should return a `nil` result to
indicate that Revel should take no action.

## Render

Called within an action "Controller.Action",
[`mvc.Controller.Render`](../docs/godoc/mvc.html#Controller.Render) does two things:
1. Adds all arguments to the controller's RenderArgs, using their local identifier as the key.
2. Executes the template "views/Controller/Action.html", passing in the controller's "RenderArgs" as the data map.

If unsuccessful (e.g. it could not find the template), it returns an ErrorResult instead.

This allows the developer to write:

<pre class="prettyprint lang-go">
func (c MyApp) Action() rev.Result {
	myValue := calculateValue()
	return c.Render(myValue)
}
</pre>

and to use "myValue" in their template.  This is usually more convenient than
constructing an explicit map, since in many cases the data will need to be
handled as a local variable anyway.

**Note:** Revel looks at the calling method name to determine the Template
  path and to look up the argument names.  Therefore, c.Render() may only be
  called from Actions.


## RenderJson

The application may call
[`RenderJson`](../docs/godoc/mvc.html#Controller.RenderJson) and pass in any Go
type (usually a struct).  Revel will serialize it using
[`json.Marshal`](http://www.golang.org/pkg/encoding/json/#Marshal) (or
MarshalIndent in development).

## Redirect

A helper function is provided for generating redirects.  It may be used in two ways.

1. Redirect to an action with no arguments:
<pre class="prettyprint lang-go">
	return c.Redirect(Hotels.Settings)
</pre>
This form is useful as it provides a degree of type safety and independence from
the routing.  (It generates the URL automatically.)

2. Redirect to a formatted string:
<pre class="prettyprint lang-go">
	return c.Redirect("/hotels/%d/settings", hotelId)
</pre>
This form is necessary to pass arguments.

It returns a 302 (Temporary Redirect) status code.


**TODO:** Status codes
