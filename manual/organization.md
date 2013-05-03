---
title: Organization
layout: manual
---


Revel requires itself and the user application to be installed into a GOPATH layout as prescribed by the go command line tool.  (See "GOPATH Environment Variable" in the [go command documentation](http://golang.org/cmd/go/))

## Example layout

Here is the default layout of a Revel application called `sample`, within a
typical Go installation.

	gocode                  GOPATH root
	  src                   GOPATH src directory
	    revel               Revel source code
	      ...
	    sample              App root
	      app               App sources
	        controllers     App controllers
	          init.go       Interceptor registration
	        models          App domain models
	        routes          Reverse routes (generated code)
	        views           Templates
	      tests             Test suites
	      conf              Configuration files
	        app.conf        Main configuration file
	        routes          Routes definition
	      messages          Message files
	      public            Public assets
	        css             CSS files
	        js              Javascript files
	        images          Image files


## The app/ directory

The `app` directory contains the source code and templates for your application.
- `app/controllers`
- `app/models`
- `app/views`

Revel requires:
- All templates are under `app/views`
- All controllers are under `app/controllers`

Beyond that, the application may organize its code however it wishes.  Revel
will watch all directories under `app/` and rebuild the app when it
notices any changes.  Any dependencies outside of `app/` will not be watched for
changes -- it is the developer's responsibility to recompile when necessary.

Additionally, Revel will import any packages within `app/` (or imported
[modules](modules.html)) that contain `init()` functions on startup, to ensure
that all of the developer's code is initialized.

The `controllers/init.go` file is a conventional location to register all of the
[interceptor](interceptors.html) hooks.  The order of `init()` functions is
undefined between source files from the same package, so collecting all of the
interceptor definitions into the same file allows the developer to specify (and
know) the order in which they are run.  (It could also be used for other
order-sensitive initialization in the future.)

## The conf/ directory

The `conf` directory contains the application's configuration files. There are
two main configuration files:

- `app.conf`, the main configuration file for the application, which contains
  standard configuration parameters
- `routes`, the routes definition file.

## The messages/ directory

The `messages` directory contains all localized message files.

## The public/ directory

Resources stored in the `public` directory are static assets that are served
directly by the Web server.  Typically it is split into three standard
sub-directories for images, CSS stylesheets and JavaScript files.

The names of these directories may be anything; the developer need only update
the routes.
