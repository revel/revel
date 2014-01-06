---
title: Templates
layout: manual
---

Revel uses [Go Templates](http://www.golang.org/pkg/text/template/).  It
searches two directories for templates:
* The application's `views` directory (and all subdirectories)
* Revel's own `templates` directory.

Given a controller named `Hello` with an action named `World`, Revel will
look for a template file named `views/Hello/World.html`. Template file names
are case insensitive so `views/hello/world.html` will work the same as
`views/HeLlO/wOrLd.HtMl`.

Revel provides templates for error pages (that display the friendly compilation
errors in DEV mode), but the application may override them by creating a
template of the equivalent name, e.g. `app/views/errors/500.html`.

## Render Context

Revel executes the template using the RenderArgs data map.  Aside from
application-provided data, Revel provides the following entries:

* "errors" - the map returned by
  [`Validation.ErrorMap`](../docs/godoc/validation.html#Validation.ErrorMap)
* "flash" - the data flashed by the previous request.

## Template Functions

Go provides
[a few functions](http://www.golang.org/pkg/text/template/#Functions) for use in
your templates.  Revel adds to those.  Read the documentation below or
[check out their source code](../docs/godoc/template.html#variables).

### eq

A simple "a == b" test.

Example:

{% raw %}

	<div class="message {{if eq .User "you"}}you{{end}}">

{% endraw %}

### set

Set a variable in the given context.

Example:

{% raw %}

	{{set . "title" "Basic Chat room"}}

	<h1>{{.title}}</h1>

{% endraw %}

### append

Add a variable to an array, or start an array, in the given context.

Example:

{% raw %}

	{{append . "moreScripts" "js/jquery-ui-1.7.2.custom.min.js"}}

    {{range .moreStyles}}
      <link rel="stylesheet" type="text/css" href="/public/{{.}}">
    {{end}}

{% endraw %}

### field

A helper for input fields.

Given a field name, it returns a struct containing the following members:
* Id: the field name, converted to be suitable as a HTML element ID.
* Name: the field name
* Value: the value of the field in the current RenderArgs
* Flash: the flashed value of the field.
* Error: the error message, if any is associated with this field.
* ErrorClass: the raw string "hasError", if there was an error, else "".

[See godoc.](../docs/godoc/field.html)

Example:

{% raw %}

	{{with $field := field "booking.CheckInDate" .}}
	  <p class="{{$field.ErrorClass}}">
	    <strong>Check In Date:</strong>
	    <input type="text" size="10" name="{{$field.Name}}" class="datepicker" value="{{$field.Flash}}">
	    * <span class="error">{{$field.Error}}</span>
	  </p>
	{{end}}

{% endraw %}

### option

Assists in constructing HTML `option` elements, in conjunction with the field
helper.

Example:

{% raw %}

	{{with $field := field "booking.Beds" .}}
	<select name="{{$field.Name}}">
	  {{option $field "1" "One king-size bed"}}
	  {{option $field "2" "Two double beds"}}
	  {{option $field "3" "Three beds"}}
	</select>
	{{end}}

{% endraw %}

### radio

Assists in constructing HTML radio `input` elements, in conjunction with the field
helper.

Example:

{% raw %}

	{{with $field := field "booking.Smoking" .}}
	  {{radio $field "true"}} Smoking
	  {{radio $field "false"}} Non smoking
	{{end}}

{% endraw %}

### nl2br

Convert newlines to HTML breaks.

Example:

{% raw %}

    You said:
    <div class="comment">{{nl2br .commentText}}</div>

{% endraw %}

### pluralize

A helper for correctly pluralizing words.

Example:

{% raw %}

	There are {{.numComments}} comment{{pluralize (len comments) "" "s"}}

{% endraw %}

### raw

Prints raw, unescaped, text.

Example:

{% raw %}

	<div class="body">{{raw .blogBody}}</div>

{% endraw %}

## Including

Go Templates allow you to compose templates by inclusion.  For example:

{% raw %}

	{{template "header.html" .}}

{% endraw %}

There are two things to note:
* Paths are relative to `app/views`

## Tips

The sample applications included with Revel try to demonstrate effective use of
Go Templates.  In particular, please take a look at:
* `revel/samples/booking/app/views/header.html`
* `revel/samples/booking/app/views/Hotels/Book.html`

It takes advantage of the helper functions to set the title and extra styles in
the template itself.

For example, the header looks like this:

{% raw %}

	<html>
	  <head>
	    <title>{{.title}}</title>
	    <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
	    <link rel="stylesheet" type="text/css" media="screen" href="/public/css/main.css">
	    <link rel="shortcut icon" type="image/png" href="/public/img/favicon.png">
	    {{range .moreStyles}}
	      <link rel="stylesheet" type="text/css" href="/public/{{.}}">
	    {{end}}
	    <script src="/public/js/jquery-1.3.2.min.js" type="text/javascript" charset="utf-8"></script>
	    <script src="/public/js/sessvars.js" type="text/javascript" charset="utf-8"></script>
	    {{range .moreScripts}}
	      <script src="/public/{{.}}" type="text/javascript" charset="utf-8"></script>
	    {{end}}
	  </head>

{% endraw %}

And templates that include it look like this:

{% raw %}

	{{set . title "Hotels"}}
	{{append . "moreStyles" "ui-lightness/jquery-ui-1.7.2.custom.css"}}
	{{append . "moreScripts" "js/jquery-ui-1.7.2.custom.min.js"}}
	{{template "header.html" .}}

{% endraw %}

## Custom Functions

Applications may register custom functions to use in templates.

Here is an example:

{% raw %}
<pre class="prettyprint lang-go">
func init() {
	revel.TemplateFuncs["eq"] = func(a, b interface{}) bool { return a == b }
}
</pre>
{% endraw %}
