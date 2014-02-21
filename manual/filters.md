---
title: Filters
layout: manual
---

Filters are the middleware -- they are individual functions that make up the
request processing pipeline.  They execute all of the framework's functionality.

The filter type is a simple function:

<pre class="prettyprint lang-go">
type Filter func(c *Controller, filterChain []Filter)
</pre>

Each filter is responsible for pulling the next filter off of the filter chain
and invoking it.  Here is the default filter stack:

<pre class="prettyprint lang-go">
// Filters is the default set of global filters.
// It may be set by the application on initialization.
var Filters = []Filter{
	PanicFilter,             // Recover from panics and display an error page instead.
	RouterFilter,            // Use the routing table to select the right Action
	FilterConfiguringFilter, // A hook for adding or removing per-Action filters.
	ParamsFilter,            // Parse parameters into Controller.Params.
	SessionFilter,           // Restore and write the session cookie.
	FlashFilter,             // Restore and write the flash cookie.
	ValidationFilter,        // Restore kept validation errors and save new ones from cookie.
	I18nFilter,              // Resolve the requested language
	InterceptorFilter,       // Run interceptors around the action.
	ActionInvoker,           // Invoke the action.
}
</pre>

## Filter chain configuration

### Global configuration

Applications may configure the filter chain by re-assigning the `revel.Filters`
variable in `init()` (by default this will be in `app/init.go` for newly
generated apps).

<pre class="prettyprint lang-go">
func init() {
	// Filters is the default set of global filters.
	revel.Filters = []Filter{
		PanicFilter,             // Recover from panics and display an error page instead.
		RouterFilter,            // Use the routing table to select the right Action
		FilterConfiguringFilter, // A hook for adding or removing per-Action filters.
		ParamsFilter,            // Parse parameters into Controller.Params.
		SessionFilter,           // Restore and write the session cookie.
		FlashFilter,             // Restore and write the flash cookie.
		ValidationFilter,        // Restore kept validation errors and save new ones from cookie.
		I18nFilter,              // Resolve the requested language
		InterceptorFilter,       // Run interceptors around the action.
		ActionInvoker,           // Invoke the action.
	}
}
</pre>

Every request is sent down this chain, top to bottom.

### Per-Action configuration

Although all requests are sent down the `revel.Filters` chain, Revel also
provides a
[`FilterConfigurator`](../docs/godoc/filterconfig.html#FilterConfigurator),
which allows the developer to add, insert, or remove filter stages based on the
Action or Controller.

This functionality is implemented by the `FilterConfiguringFilter`, itself a
filter stage.

## Implementing a Filter

### Keep the chain going

Filters are responsible for invoking the next filter to continue the request
processing.  This is generally done with an expression as shown here:

<pre class="prettyprint lang-go">
var MyFilter = func(c *revel.Controller, fc []revel.Filter) {
	// .. do some pre-processing ..

	fc[0](c, fc[1:]) // Execute the next filter stage.

	// .. do some post-processing ..
}
</pre>

### Getting the app Controller type

Filters receive the base `*Controller` type as an
argument, rather than the actual Controller type that was invoked.  If your
filter requires access to the actual Controller type that was invoked, it may
grab it with the following trick:

<pre class="prettyprint lang-go">
var MyFilter = func(c *revel.Controller, fc []revel.Filter) {
	if ac, err := c.AppController.(*MyController); err == nil {
		// Have an instance of *MyController...
	}

	fc[0](c, fc[1:]) // Execute the next filter stage.
}
</pre>

Note: this pattern is frequently an indicator that
[interceptors](interceptors.html) may be a better mechanism to accomplish the
desired functionality.
