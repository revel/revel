---
title: Plugins
layout: manual
---

Plugins are types that may be registered to hook into application and request lifecycle events.

A plugin meets the following interface (and is notified for each of these events):

<pre class="prettyprint lang-go">
type Plugin interface {
	// Called on server startup (and on each code reload).
	OnAppStart()
	// Called after the router has finished configuration.
	OnRoutesLoaded(router *Router)
	// Called before every request.
	BeforeRequest(c *Controller)
	// Called after every request (except on panics).
	AfterRequest(c *Controller)
	// Called when a panic exits an action, with the recovered value.
	OnException(c *Controller, err interface{})
	// Called after every request (panic or not), after the Result has been applied.
	Finally(c *Controller)
}
</pre>

To define a Plugin of your own, declare a type that embeds `rev.EmptyPlugin`, and override just the methods that you want.  Then register it with `rev.RegisterPlugin`.

<pre class="prettyprint lang-go">
type DbPlugin struct {
	rev.EmptyPlugin
}

func (p DbPlugin) OnAppStart() {
	...
}

func init() {
	rev.RegisterPlugin(DbPlugin{})
}
</pre>

Revel will invoke all methods on the single instance provided to `RegisterPlugin`, so ensure that the methods are threadsafe.

### Areas for development

* Add more things that plugins can handle: the entire request, rendering a template, a "Finally" that gets invoked after a request regardless of whether there was a panic.
* Provide an "official" place to put calls to RegisterPlugin.
