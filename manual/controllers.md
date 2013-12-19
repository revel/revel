---
title: Overview
layout: manual
---

A **Controller** is any type that embeds `*revel.Controller` (directly or indirectly).
    
Typically:
{% raw %}
<pre class="prettyprint lang-go">
type AppController struct {
  *revel.Controller
}
</pre>
{% endraw %}

(`*revel.Controller` must be embedded as the first type in your struct)

The `revel.Controller` is the context for the request.  It contains the request
and response data.  Please refer to [the godoc](../docs/godoc/controller.html)
for the full story, but here is the definition (along with definitions of helper types):

{% raw %}
<pre class="prettyprint lang-go">
type Controller struct {
    Name          string          // The controller name, e.g. "Application"
    Type          *ControllerType // A description of the controller type.
    MethodType    *MethodType     // A description of the invoked action type.
    AppController interface{}     // The controller that was instantiated.

    Request  *Request
    Response *Response
    Result   Result

    Flash      Flash                  // User cookie, cleared after 1 request.
    Session    Session                // Session, stored in cookie, signed.
    Params     *Params                // Parameters from URL and form (including multipart).
    Args       map[string]interface{} // Per-request scratch space.
    RenderArgs map[string]interface{} // Args passed to the template.
    Validation *Validation            // Data validation helpers
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
{% endraw %}
As part of handling a HTTP request, Revel instantiates an instance of your
Controller, and it sets all of these properties on the embedded
`revel.Controller`.  Therefore, Revel does not share Controller instances between
requests.

