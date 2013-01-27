---
title: Deployment
layout: manual
---

## Overview

There are a couple common deployment routes:

* Build the app locally and copy it to the server.
* On the server, pull the updated code, build it, and run it.
* Use Heroku to manage deployment.

The command line sessions demonstrate interactive deployment -- typically one
would use a tool for daemonizing their web server.  Common tools:

* [Ubuntu Upstart](http://upstart.ubuntu.com)
* [systemd](http://www.freedesktop.org/wiki/Software/systemd)

## Build locally

Revel apps may be deployed to machines that do not have a functioning Go
installation.  The [command line tool](tool.html) provides the `package` command
which compiles and zips the app, along with a script to run it.

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
    $ unzip app.zip
	$ bash run.sh

Presently there is no explicit cross-compilation support, so this only works if
you develop and deploy to the same architecture, or if you configure your go
installation to build to the desired architecture by default.

### Incremental deployment

Since a statically-linked binary with a full set of assets can grow to be quite
large, incremental deployment is supported.

    # Build the app into a temp directory
    $ revel build import/path/to/app /tmp/app

    # Rsync that directory into the home directory on the server
    $ rsync -vaz --rsh="ssh" /tmp/app server

    # Connect to server and restart the app.
    ...

Rsync has full support for copying over ssh.  For example, here's a more complicated connection.

    # A more complicated example using custom certificate, login name, and target directory
    $ rsync -vaz --rsh="ssh -i .ssh/go.pem" /tmp/myapp2 ubuntu@ec2-50-16-80-4.compute-1.amazonaws.com:~/rsync


## Build on the server

This method relies on your version control system to distribute updates.  It
requires your server to have a Go installation.  In return, it allows you to
avoid potentially having to cross-compile.

    $ ssh server
    ... install go ...
    ... configure your app repository ...

    # Move to the app directory (in your GOPATH), pull updates, and run the server.
    $ cd gocode/src/import/path/to/app
    $ git pull
    $ revel run import/path/to/app prod

## Heroku

**jamesward** kindly made a
  [Heroku buildpack for Revel apps](https://github.com/robfig/heroku-buildpack-go-revel).
  He also wrote
  [a blog post about getting a sample app up and running on Heroku](http://www.jamesward.com/2012/09/28/run-revel-apps-on-heroku).


## Areas for development:

* Cross-compilation (e.g. develop on OSX, deploy on Linux).
