---
title: Mixing in
layout: manual
---

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

In this example, the developer has created a `Controller` with the behaviors they
want to use across much of their application, and they can mix it in by
embedding that `Controller` within others.  Here, the interceptors will run and
the actions on `AppController` will have access to the `*mgo.Session`.
