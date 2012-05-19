# Revel!

This is a port of the amazing [Play! framework](http://www.playframework.org) to the [Go language](http://www.golang.org).

It is a high productivity web framework

[![Build Status](https://secure.travis-ci.org/robfig/revel.png?branch=master)](http://travis-ci.org/robfig/revel)

# Simple Example

Write a routes file declaration for some actions, assets and a catchall:

```
# conf/routes
GET /                       Application.Index
GET /app/{id}               Application.ShowApp
GET /app/{id}/{action}      Application.{action}
GET /public/                staticDir:public
*   /{controller}/{action}  {controller}.{action}
```

Declare a Controller:

```go
// app/controllers/app.go
package controllers
import "github.com/robfig/revel"

type Application struct {
	*rev.Controller
}

func (c Application) Index() rev.Result {
	return c.Render()
}

func (c Application) ShowApp(id int) rev.Result {
	return c.Render(id)
}
```

Define a view using [go templates](http://www.golang.org/pkg/text/template/):

```
{{/* app/views/Application/ShowApp.html */}}
{{template "header.html" .}}
This is app {{.id}}!
{{template "footer.html" .}}
```

# Bigger Example

This is an example Controller method that processes a Login request.  It demonstrates:

- Validating posted data
- Keeping validation errors and parameters in the Flash scope (a cookie that lives for one page view)
- Redirecting
- Setting a cookie

```go
func (c Login) DoLogin(username, password string) rev.Result {
	// Validate parameters.
	c.Validation.Required(username).Message("Please enter a username.")
	c.Validation.Required(password).Message("Please enter a password.")
	c.Validation.Required(len(password) > 6).Message("Password must be at least 6 chars.")

	// If validation failed, redirect back to the login form.
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Login.ShowLogin)
	}

 	// Check the credentials.
	if username != "user" || password != "password" {
		c.Flash.Error("Username or password not recognized")
		c.FlashParams()
		return c.Redirect(Login.ShowLogin)
	}

	// Success.  Set the login cookie.
	c.SetCookie(&http.Cookie{
		Name:    "Login",
		Value:   "Success",
		Path:    "/",
		Expires: time.Now().AddDate(0, 0, 7),
	})
	c.Flash.Success("Login successful.")

	return c.Redirect(Application.Index)
}
```

There are also helpers to make validation errors easy to surface in the template.  Here's an example from the "register a new user" form in the sample application:

```html
{{template "header.html" .}}

<h1>Register:</h1>

<form action="{{url "Application.SaveUser"}}" method="POST">
  {{with $field := field "user.Username" .}}
    <p class="{{$field.ErrorClass}}">
      <strong>Username:</strong>
      <input type="text" name="{{$field.Name}}" size="16" value="{{$field.Value}}"> *
      <span class="error">{{$field.Error}}</span>
    </p>
  {{end}}
  {{with $field := field "user.Password" .}}
    <p class="{{$field.ErrorClass}}">
      <strong>Password:</strong> <input type="password" name="{{$field.Name}}" size="16" value="{{$field.Value}}"> *
      <span class="error">{{$field.Error}}</span>
    </p>
  {{end}}
  <p class="buttons">
    <input type="submit" value="Register"> <a href="{{url "Application.Index"}}">Cancel</a>
  </p>
</form>
```


# Quick start

From your GOPATH base:

```
go get github.com/howeyc/fsnotify \
  github.com/kless/goconfig/config \
  github.com/robfig/revel
go build -o /bin/rev github.com/robfig/revel/cmd
./bin/rev github.com/robfig/revel/samples/booking
```

# How it works

- The command line tool runs a harness that acts as a reverse proxy.
- It listens on port 9000 and watches the app files for changes.
- It forwards requests to the running server.  If the server isn't running or a source file has changed since the last request, it rebuilds the app.
- If it needs to rebuild the app, the harness analyzes the source code and produces a `app/tmp/main.go` file that contains all of the meta information necessary required to support the various magic as well as runs the real app server.
- It uses `go build` to compile the app.  If there is a compile error, it shows a helpful error page to the user as the response.
- If the app compiled successfully, it runs the app and forwards the request when it detects that the app server has finished starting up.

# Features

## Implemented

The basic workflow is already working, as you can see in the sample app.

- Hot code reload
- Compile error pages for code / templates
- Controller model
- Template rendering (Go Templates)
- Routing
- Reverse routing
- Static file serving
- Basic validation
- Flash Scope
- Data binding
- Interceptors
- Form validation
- Session (signed cookie)
- application.conf
- "Production mode"
- Handle panics -- it shows the bottom line of app source in the trace.
- Websockets

## TODO

There is a large list of things left to do.

- SSL
- Render other content types.
- Return different return codes.
- app/views/errors/{404,500}.html
- Jobs
- Plugins
- Modules
- Compiled assets (e.g. LESS, Coffee)
- Logging
- ORM?
- Alternate template languages
- Internationalization support
- Testing tools
- Command line tool support for initializing the default project layout.
- Performance testing / tuning
- AppEngine support
- HOCON library for application.conf
- .routes.Application.Method
- Multipart forms / file uploads
- Extract default error messages to resource file
- Make it not hard to do this: https://gist.github.com/2328236
- BigDecimal replacement (for currency / that works with MySQL)
- Fixtures
- How to do moreStyles / moreScripts equivalent.
- Reflection magic does not work if app code imports revel with a modified identifier.

# File layout

Revel depends on the GOPATH layout prescribed by the go command line tool.  A description is at the end.

Note that Revel must be installed in the same GOPATH as the user app -- it uses that assumption to find its own source, e.g. for finding templates.


## Example layout

Here is the default layout of a Revel application called `sample`, within a
typical Go installation.

```
gocode                         GOPATH root
  src                          GOPATH src directory
    revel                      Revel source code
    sample
      app                      App sources
        controllers            App controllers
        models                 App business layer
        views                  Templates
      conf                     Configuration files
        application.conf       Main configuration file
        routes                 Routes definition
      public                   Public assets
        css                    CSS files
        js                     Javascript files
        img                    Image files
```

## The app/ directory

The `app` directory contains the source code and templates for your application.

By default, these are:

- `app/controllers`
- `app/models`
- `app/views`

You can add any directories you wish for .go code, but all templates must be in the views directory.

## The public/ directory

Resources stored in the `public` directory are static assets that are served directly by the Web server.  Typically it is split into three standard sub-directories for images, CSS stylesheets and JavaScript files.

## The conf/ directory

The `conf` directory contains the application's configuration files. There are two main configuration files:

- `application.conf`, the main configuration file for the application, which contains standard configuration parameters (not yet implemented)
- `routes`, the routes definition file.

# Appendix

## The GOPATH Environment Variable

from http://golang.org/cmd/goinstall/

GOPATH may be set to a colon-separated list of paths inside which Go code, package objects, and executables may be found.

Set a GOPATH to use goinstall to build and install your own code and external libraries outside of the Go tree (and to avoid writing Makefiles).

The top-level directory structure of a GOPATH is prescribed:

The 'src' directory is for source code. The directory naming inside 'src' determines the package import path or executable name.

The 'pkg' directory is for package objects. Like the Go tree, package objects are stored inside a directory named after the target operating system and processor architecture ('pkg/$GOOS_$GOARCH'). A package whose source is located at '$GOPATH/src/foo/bar' would be imported as 'foo/bar' and installed as '$GOPATH/pkg/$GOOS_$GOARCH/foo/bar.a'.

The 'bin' directory is for executable files. Goinstall installs program binaries using the name of the source folder. A binary whose source is at 'src/foo/qux' would be built and installed to '$GOPATH/bin/qux'. (Note 'bin/qux', not 'bin/foo/qux' - this is such that you can put the bin directory in your PATH.)

Here's an example directory layout:

```
GOPATH=/home/user/gocode

/home/user/gocode/
	src/foo/
		bar/               (go code in package bar)
		qux/               (go code in package main)
	bin/qux                    (executable file)
	pkg/linux_amd64/foo/bar.a  (object file)
```

