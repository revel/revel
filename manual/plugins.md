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

To define a Plugin of your own, declare a type that embeds `revel.EmptyPlugin`,
and override just the methods that you want.  Then register it with
`revel.RegisterPlugin`.

<pre class="prettyprint lang-go">
type DbPlugin struct {
	revel.EmptyPlugin
}

func (p DbPlugin) OnAppStart() {
	...
}

func init() {
	revel.RegisterPlugin(DbPlugin{})
}
</pre>

Revel will invoke all methods on the single instance provided to
`RegisterPlugin`, so ensure that the methods are threadsafe.

One limitation of Plugins is that they receive the base `*Controller` type as an
argument, rather than the actual Controller type that was invoked.  If your
plugin requires access to the actual Controller type that was invoked, it may
grab it with the following trick:

<pre class="prettyprint lang-go">
func (p DbPlugin) BeforeRequest(c *revel.Controller) revel.Result {
	ac, err := c.AppController.(*MyController)
	if err != nil {
		return nil  // Not the desired controller type
	}

	// Have an instance of *MyController
}
</pre>

Note: this pattern is frequently an indicator that
[interceptors](interceptors.html) may be a better mechanism to accomplish the
desired functionality.

### Areas for development

* Add more things that plugins can handle: the entire request, rendering a template, etc.
* Provide an "official" place to put calls to RegisterPlugin.
