---
title: Sharing functionality
layout: manual
---

Revel provides ways to share functionality across different scopes:

* actions
* controllers
* applications

## Actions

Related actions share a single Controller type.  The developer may add data
fields to it in order to store more context about the request.
[Interceptors](interceptors.html) may be registered to run before or after any
actions on a Controller.

## Controllers

Revel allows mixing Controllers together in order to share fields, methods, and
interceptors across multiple Controller types.

Here is an example:

<pre class="prettyprint lang-go">
// MongoController provides access to our MongoDB
type MongoController struct {
	*revel.Controller
	Session *mgo.Session
}

func (c *MongoController) Begin() revel.Result {
	c.Session = ...
}

func init() {
	revel.InterceptMethod((*MongoController).Begin, revel.BEFORE)
}

type AppController struct {
	*revel.Controller
	MongoController
}
</pre>

In this example, the developer has created a `MongoController` with the
behaviors they want to use across much of their application, and they can mix it
in by embedding that `MongoController` within others.  Here, the interceptors
will run and the actions on `AppController` will have access to the
`*mgo.Session`.

## Applications

[Modules](modules.html) may be used to share controller types, templates, and
assets across applications.
