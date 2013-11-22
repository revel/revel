---
title: Frequently Asked Questions
layout: manual
---

> How do I integrate existing http.Handlers with Revel?

As shown in the [concept diagram](concepts.html), the http.Handler is where Go
hands off the user's request for processing.  Revel's handler is extraordinarily
simple -- it just creates the Controller instance and passes the request to the
Filter Chain.

Applications may integrate existing http.Handlers by overriding the default
Handler:

<pre class="prettyprint lang-go">{% capture guy %}{% raw %}
func installHandlers() {
	var (
		serveMux     = http.NewServeMux()
		revelHandler = revel.Server.Handler
	)
	serveMux.Handle("/",     revelHandler)
	serveMux.Handle("/path", myHandler)
	revel.Server.Handler = serveMux
}

func init() {
	revel.OnAppStart(installHandlers)
}{% endraw %}{% endcapture %}{{ guy|escape }}
</pre>


> What is the relationship between interceptors, filters, and modules?

1. Modules are packages that can be plugged into an application. They allow
sharing of controllers, views, assets, and other code between multiple Revel
applications (or from third-party sources).

2. Filters are functions that may be hooked into the request processing
pipeline.  They generally apply to the application as a whole and handle
technical concerns, orthogonal to application logic.

3. Interceptors are a convenient way to package data and behavior, since
embedding a type imports its interceptors and fields.  This makes interceptors
useful for things like verifying the login cookie and saving that information
into a field.  Interceptors can be applied to one or more controllers.


