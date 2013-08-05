---
title: Results
layout: manual
---

Actions must return a [`revel.Result`](../docs/godoc/results.html#Result), which
handles the response generation.  It adheres to a simple interface:

<pre class="prettyprint lang-go">
type Result interface {
	Apply(req *Request, resp *Response)
}
</pre>

[`revel.Controller`](../docs/godoc/controller.html#Controller) provides a couple
methods to produce Results:
* Render, RenderTemplate - render a template, passing arguments.
* RenderJson, RenderXml - serialize a structure to json or xml.
* RenderText - return a plaintext response.
* Redirect - redirect to another action or URL
* RenderFile - return a file, generally to be downloaded as an attachment.
* RenderError - return a 500 response that renders the errors/500.html template.
* NotFound - return a 404 response that renders the errors/404.html template.
* Todo - return a stub response (500)

Additionally, the developer may define their own `revel.Result` and return that.

### Setting the Status Code / Content Type

Each built-in Result has a default Status Code and Content Type.  To override
those defaults, simply set those properties on the response:

<pre class="prettyprint lang-go">
func (c App) Action() revel.Result {
	c.Response.Status = http.StatusTeapot
	c.Response.ContentType = "application/dishware"
	return c.Render()
}
</pre>

## Render

Called within an action (e.g. "Controller.Action"),
[`mvc.Controller.Render`](../docs/godoc/controller.html#Controller.Render) does two things:
1. Adds all arguments to the controller's RenderArgs, using their local identifier as the key.
2. Executes the template "views/Controller/Action.html", passing in the controller's "RenderArgs" as the data map.

If unsuccessful (e.g. it could not find the template), it returns an ErrorResult instead.

This allows the developer to write:

<pre class="prettyprint lang-go">
func (c MyApp) Action() revel.Result {
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


## RenderJson / RenderXml

The application may call
[`RenderJson`](../docs/godoc/controller.html#Controller.RenderJson) or
[`RenderXml`](../docs/godoc/controller.html#Controller.RenderXml) and pass in any Go
type (usually a struct).  Revel will serialize it using
[`json.Marshal`](http://www.golang.org/pkg/encoding/json/#Marshal) or
[`xml.Marshal`](http://www.golang.org/pkg/encoding/xml/#Marshal).

If `results.pretty=true` in `app.conf`, serialization will be done using
`MarshalIndent` instead, to produce nicely indented output for human
consumption.

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

## Adding your own Result

Here is an example of adding a simple Result.

Create this type:

<pre class="prettyprint lang-go">
type Html string

func (r Html) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/html")
	resp.Out.Write([]byte(r))
}
</pre>

Then use it in an action:

<pre class="prettyprint lang-go">{% capture html %}
func (c *App) Action() revel.Result {
	return Html("<html><body>Hello World</body></html>")
}{% endcapture %}{{ html|escape }}
</pre>

## Status Codes

Each Result will set a status code by default.  You can override the default
status code by setting one yourself:

<pre class="prettyprint lang-go">
func (c *App) CreateEntity() revel.Result {
	c.Response.Status = 201
	return c.Render()
}
</pre>
