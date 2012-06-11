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
	        models          App domain models
	        views           Templates
	      conf              Configuration files
	        app.conf        Main configuration file
	        routes          Routes definition
	      public            Public assets
	        css             CSS files
	        js              Javascript files
	        img             Image files


## The app/ directory

The `app` directory contains the source code and templates for your application.
- `app/controllers`
- `app/models`
- `app/views`

Revel requires:
- All templates are under `app/views`
- All controllers are under `app/controllers`

Beyond that, the application may organize its code however it wishes.  Revel
will watch all directories under `app/` for changes and rebuild the app when it
notices any changes.  Any dependencies outside of `app/` will not be watched for
changes -- it is the developer's responsibility to recompile when necessary.

## The public/ directory

Resources stored in the `public` directory are static assets that are served
directly by the Web server.  Typically it is split into three standard
sub-directories for images, CSS stylesheets and JavaScript files.

The names of these directories may be anything; the developer need only update
the routes.

## The conf/ directory

The `conf` directory contains the application's configuration files. There are
two main configuration files:

- `app.conf`, the main configuration file for the application, which contains
  standard configuration parameters
- `routes`, the routes definition file.
