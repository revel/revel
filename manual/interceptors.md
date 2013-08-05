---
title: Interceptors
layout: manual
---

An "interceptor" is a function that is invoked by the framework before or after an action invocation.  It allows a form of
[Aspect Oriented Programming](http://en.wikipedia.org/wiki/Aspect-oriented_programming),
which is useful for some common concerns:
* Request logging
* Error handling
* Stats keeping

In Revel, an interceptor can take one of two forms:

1. Func Interceptor: A function meeting the
   [`InterceptorFunc`](../docs/godoc/intercept.html#InterceptorFunc) interface.
	* Does not have access to specific application Controller invoked.
	* May be applied to any / all Controllers in an application.

2. Method Interceptor: A controller method accepting no arguments and returning a `revel.Result`
	* May only intercept calls to the bound Controller.
	* May modify the invoked controller as desired.

Interceptors are called in the order that they are added.

## Intercept Times

An interceptor can be registered to run at four points in the request lifecycle:

1. BEFORE: After the request has been routed, the session, flash, and parameters decoded, but before the action has been invoked.
2. AFTER: After the request has returned a Result, but before that Result has been applied.  These interceptors are not invoked if the action panicked.
3. PANIC: After a panic exits an action or is raised from applying the returned Result.
4. FINALLY: After an action has completed and the Result has been applied.

## Results

Interceptors typically return `nil`, in which case the request continues to
be processed without interruption.

The effect of returning a non-`nil` `revel.Result` depends on when the interceptor
was invoked.

1. BEFORE:  No further interceptors are invoked, and neither is the action.
2. AFTER: All interceptors are still run.
3. PANIC: All interceptors are still run.
4. FINALLY: All interceptors are still run.

In all cases, any returned Result will take the place of any existing Result.

In the BEFORE case, however, that returned Result is guaranteed to be final,
while in the AFTER case it is possible that a further interceptor could emit its
own Result.

## Examples

### Func Interceptor

Here's a simple example defining and registering a Func Interceptor.

<pre class="prettyprint lang-go">{% capture guy %}{% raw %}
func checkUser(c *revel.Controller) revel.Result {
	if user := connected(c); user == nil {
		c.Flash.Error("Please log in first")
		return c.Redirect(App.Index)
	}
	return nil
}

func init() {
	revel.InterceptFunc(checkUser, revel.BEFORE, &Hotels{})
}{% endraw %}{% endcapture %}{{ guy|escape }}
</pre>

### Method Interceptor

A method interceptor signature may have one of these two forms:

{% raw %}
<pre class="prettyprint lang-go">
func (c AppController) example() revel.Result
func (c *AppController) example() revel.Result
</pre>
{% endraw %}

Here's the same example that operates only on the app controller.

{% raw %}
<pre class="prettyprint lang-go">
func (c Hotels) checkUser() revel.Result {
	if user := connected(c); user == nil {
		c.Flash.Error("Please log in first")
		return c.Redirect(App.Index)
	}
	return nil
}

func init() {
	revel.InterceptMethod(Hotels.checkUser, revel.BEFORE)
}
</pre>
{% endraw %}
