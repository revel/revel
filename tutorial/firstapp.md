---
title: The "Hello World" app
layout: tutorial
---

This article runs through the quick exercise of implementing the "Hello World"
application from
[the Play! example](http://www.playframework.org/documentation/1.2.4/firstapp).

Let's start with the **myapp** project that [we created previously](createapp.html).

Edit the **app/views/Application/Index.html** template to add this form:

	<form action="/Application/Hello" method="GET">
	    <input type="text" name="myName" />
	    <input type="submit" value="Say hello!" />
	</form>

Refresh the page to see our work.

![The Say Hello form](../img/AlohaForm.png)

Let's try submitting that form.

![Route not found](../img/HelloRouteNotFound.png)

That makes sense.  Add the action to **app/controllers/app.go**:

	func (c Application) Hello(myName string) rev.Result {
		return c.Render(myName)
	}


Next, we have to create the view.  Create a file
**app/views/Application/Hello.html**, with this content:

	{{set "title" "Home" .}}
	{{template "header.html" .}}

	<h1>Hello {{.myName}}</h1>
	<a href="/">Back to form</a>

	{{template "footer.html" .}}

Refresh the page, and you should see a greeting:

![Hello Robfig](../img/HelloRobfig.png)

Lastly, let's add some validation.  The name should be required, and at least
three characters.

To do this, let's use the [validation module](../manual/validation.html).  Edit
your action in **app/controllers/app.go**:

	func (c Application) Hello(myName string) rev.Result {
		c.Validation.Required(myName).Message("Your name is required!")
		c.Validation.MinSize(myName, 3).Message("Your name is not long enough!")

		if c.Validation.HasErrors() {
			c.Validation.Keep()
			c.FlashParams()
			return c.Redirect(Application.Index)
		}

		return c.Render(myName)
	}

Now it will send the user back to `Index()` if they have not entered a valid
name. Their name and the validation error are kept in the
[Flash](../manual/sessionflash.html), which is a temporary cookie.

Let's use that data in the form.  Edit **app/views/Application/Index.html**:

{% literal %}

	<h1>Aloha World</h1>

	{{range .errors}}
		<p style="color:#c00">
			{{.Message}}
		</p>
	{{end}}

	<form action="/Application/Hello" method="GET">
		<input type="text" name="myName" value="{{.flash.myName}}" />
		<input type="submit" value="Say hello!" />
	</form>

{% endliteral %}

Now when we submit a single letter as our name:

![Example error](../img/HelloNameNotLongEnough.png)

Success, we got an appropriate error and our input was saved for us to edit.
