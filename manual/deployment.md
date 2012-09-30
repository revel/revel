---
title: Deployment
layout: manual
---

## SCP

Revel apps may be deployed to machines that do not have a functioning Go
installation.  The [command line tool](tool.html) provides the `package` command
which compiles and zips the app, along with a script to run it.

A typical deployment would look like this:

	# Run and test my app.
	$ revel run import/path/to/app
	.. test app ..

	# Package it up.
	$ revel package import/path/to/app
	Your archive is ready: app.zip

	# Copy to the target machine.
	$ scp app.zip target:/srv/

	# Run it on the target machine.
	$ ssh target
	$ cd /srv/
	$ ./run.sh

Areas for development:

* Cross-compilation (e.g. develop on OSX, deploy on Linux).

## Heroku

**jamesward** kindly made a
  [Heroku buildpack for Revel apps](https://github.com/jamesward/heroku-buildpack-go-revel).
  He also wrote
  [a blog post about getting a sample app up and running on Heroku](http://www.jamesward.com/2012/09/28/run-revel-apps-on-heroku).
