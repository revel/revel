# Go Play!

This is a port of the amazing [Play! framework](http://www.playframework.org) to the [Go language](http://www.golang.org).

It tries to be a high-productivity web framework.

# Example

1. Write a routes file declaration for some actions, assets and a catchall:
```
conf/routes:

GET /                     Application.Index
GET /app/{id}             Application.ShowApp
GET /public/              staticDir:public
* /{controller}/{action} {controller}.{action}
```

2. Declare a Controller:

```go
app/controllers/app.go:

package controllers
import "play"

type Application struct {
	*play.Controller
}

func (c *Application) ShowApp(id int) play.Result {
	return c.Render(id)
}
```

3. Define a view using [go templates](http://www.golang.org/pkg/text/template/):

```
app/views/Application/ShowApp.html:

{{template "header.html" .}}
This is app {{.id}}!
{{template "footer.html" .}}
```

# Quick start

- Clone this repo into your GOPATH.  (If you don't know what GOPATH is, see appendix)

```
export GOPATH=/Users/$USER/gocode
mkdir -p $GOPATH/src
cd $GOPATH
git clone github.com/robfig/go-play src/play
```

- Build the `play` command line tool: `go build -o ./bin/play src/play/cmd`
- Run the sample app by invoking the tool with the import path: `./bin/play play/sample`
- Visit localhost:9000

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

## TODO

There is a large list of things left to do.

- Data binding
- Form validation
- application.conf
- Session support
- Interceptors
- Jobs
- Plugins
- Async / evented support (e.g. suspend/resume)
- Websockets
- Compiled assets (e.g. LESS, Coffee)
- Logging
- ORM?
- Alternate template languages
- Internationalization support
- Testing tools
- Command line tool support for initializing the default project layout.
- Windows support (fsnotify package is OSX/Linux only)
- Performance testing / tuning


# File layout

Go Play! depends on the GOPATH layout prescribed by the go command line tool.  A description is at the end.

Note that Play must be installed in the same GOPATH as the user app -- it uses that assumption to find its own source, e.g. for finding templates.


## Example layout

Here is the default layout of a Go Play application called `sample`, within a
typical Go installation.

```
gocode                         GOPATH root
  src                          GOPATH src directory
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
    play                       Go Play source code
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

