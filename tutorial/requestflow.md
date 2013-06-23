---
title: The Request Flow
layout: tutorial
---

In the [previous article](createapp.html) we created a new Revel application
called **myapp**. In this article we look at how Revel handles the HTTP request
to http://localhost:9000/, resulting in the welcome message.

### Routes

The first thing that Revel does is check the **conf/routes** file -- the
generated routes file contains a route:

	GET     /                                       App.Index

This tells Revel to invoke the **Index** method of the **App**
controller when it receives a **GET** request to **/**.

### Actions

Let's follow this call to the code, in **app/controllers/app.go**:

	package controllers

	import "github.com/robfig/revel"

	type App struct {
		*revel.Controller
	}

	func (c App) Index() revel.Result {
		return c.Render()
	}

All controllers must be structs that embed `*revel.Controller`
in the first slot (directly or indirectly). Any method on a controller that is
exported and returns a `revel.Result` may be treated as an Action.

The Revel controller provides many useful methods for generating Results. In
this example, it calls [`Render()`](../docs/godoc/mvc.html#Controller.Render),
which tells Revel to find and render a template as the response (**200 OK**).

### Templates

All templates are kept in the **app/views** directory. When an explicit
template name is not specified, Revel looks for a template matching the action.
In this case, Revel finds the **app/views/App/Index.html** file, and
renders it as a [Go template](http://www.golang.org/pkg/html/template).

{% raw %}

	{{set . "title" "Home"}}
	{{template "header.html" .}}

	<header class="hero-unit" style="background-color:#A9F16C">
	  <div class="container">
	    <div class="row">
	      <div class="hero-text">
	        <h1>It works!</h1>
	        <p></p>
	      </div>
	    </div>
	  </div>
	</header>

	<div class="container">
	  <div class="row">
	    <div class="span6">
	      {{template "flash.html" .}}
	    </div>
	  </div>
	</div>

	{{template "footer.html" .}}

{% endraw %}

Beyond the functions provided by the Go templates, Revel adds
[a few helpful ones](../manual/templates.html) of its own.

This template is very simple.  It:

1. Adds a new **title** variable to the render context.
2. Includes the **header.html** template (which uses the title).
3. Displays a welcome message.
4. Includes the **flash.html** template, which shows any flashed messages.
5. Includes the **footer.html**.

If you look at header.html, you can see some more template tags in action:

{% raw %}

	<!DOCTYPE html>
	<html>
	  <head>
	    <title>{{.title}}</title>
	    <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
	    <link rel="stylesheet" type="text/css" media="screen" href="/public/stylesheets/main.css">
	    <link rel="shortcut icon" type="image/png" href="/public/images/favicon.png">
	    <script src="/public/javascripts/jquery-1.5.2.min.js" type="text/javascript" charset="utf-8"></script>
	    {{range .moreStyles}}
	      <link rel="stylesheet" type="text/css" href="/public/{{.}}">
	    {{end}}
	    {{range .moreScripts}}
	      <script src="/public/{{.}}" type="text/javascript" charset="utf-8"></script>
	    {{end}}
	  </head>
	  <body>

{% endraw %}

You can see the title being set, and you can also see that it accepts JS and CSS
files included from calling templates in the **moreStyles** and **moreScripts**
variables.

### Hot-reload

Let's change the welcome message.  In **Index.html**, change

	<h1>It works!</h1>

to

	<h1>Hello World</h1>

Refresh your browser, and you should see the change immediately!  Revel noticed
that your template changed and reloaded it.

Revel watches:

* All go code under **app/**
* All templates under **app/views/**
* Your routes file: **conf/routes**

Changes to any of those will cause Revel to update your running app with the
newest code.  Try it right now: open **app/controllers/app.go** and introduce
an error.

Change

	return c.Render()

to

	return c.Renderx()

Refresh the page and Revel will display a helpful error message:

![A helpful error message](../img/helpfulerror.png)

Lastly, let's pass some data into the template.

In **app/controllers/app.go**, change:

	return c.Renderx()

to:

	greeting := "Aloha World"
	return c.Render(greeting)

In **app/views/App/Index.html**, change:

{% raw %}

	<h1>Hello World</h1>

{% endraw %}

to:

{% raw %}

	<h1>{{.greeting}}</h1>

{% endraw %}

Refresh the page and you should see your Hawaiian greeting.

![A Hawaiian greeting](../img/AlohaWorld.png)

**Next: [Make a simple Hello World application](firstapp.html).**
