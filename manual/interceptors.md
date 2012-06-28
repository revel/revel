---
title: Interceptors
layout: manual
---

An "interceptor" is a function that is invoked by the framework at a designated
time an action invcation.  It allows a form of
[Aspect Oriented Programming](http://en.wikipedia.org/wiki/Aspect-oriented_programming),
which is useful for some common concerns:
* Request logging
* Beginning and committing transactions (or rolling back in the case of an exception)
* Error handling
* Stats keeping

In Revel, an interceptor can take one of two forms:

1. Func Interceptor: A function meeting the
   [`InterceptorFunc`](../docs/godoc/intercept.html#InterceptorFunc) interface.
	* Does not have access to specific application Controller invoked.
	* May be applied to any / all Controllers in an application.

2. Method Interceptor: A controller method accepting no arguments and returning a `rev.Result`
	* May only intercept calls to the bound Controller.
	* May modify the invoked controller as desired.

Interceptors are called in the order that they are added.

## Results

Interceptors typically return `nil`, in which case they the request continues to
be processed without interruption.

The effect of returning a non-`nil` `rev.Result` depends on when the interceptor
was invoked.

1. BEFORE:  No further interceptors are invoked, and neither is the action.
2. AFTER: All interceptors are still run.

In all cases, any returned Result will take the place of any existing Result.

In the BEFORE case, however, that returned Result is guaranteed to be final,
while in the AFTER case it is possible that a further interceptor could emit its
own Result.

## Examples

### Func Interceptor

Here's a simple example defining and registering a Func Interceptor.

{% literal %}
<pre class="prettyprint lang-go">
func checkUser(c *rev.Controller) rev.Result {
	if user := connected(c); user == nil {
		c.Flash.Error("Please log in first")
		return c.Redirect(Application.Index)
	}
	return nil
}

func init() {
	rev.InterceptFunc(checkUser, rev.BEFORE, &Hotels{})
}
</pre>
{% endliteral %}

### Method Interceptor

A method interceptor signature may have one of these two forms:

{% literal %}
<pre class="prettyprint lang-go">
func (c AppController) example() rev.Result
func (c *AppController) example() rev.Result
</pre>
{% endliteral %}

Here's the same example that operates only on the app controller.

{% literal %}
<pre class="prettyprint lang-go">
func (c Hotels) checkUser() rev.Result {
	if user := connected(c); user == nil {
		c.Flash.Error("Please log in first")
		return c.Redirect(Application.Index)
	}
	return nil
}

func init() {
	rev.InterceptMethod(checkUser, rev.BEFORE)
}
</pre>
{% endliteral %}
