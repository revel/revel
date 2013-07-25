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
	Your archive is ready: app.tar.gz

	# Copy to the target machine.
	$ scp app.tar.gz target:/srv/

	# Run it on the target machine.
	$ ssh target
	$ cd /srv/
    $ tar xzvf app.tar.gz
	$ bash run.sh

This only works if you develop and deploy to the same architecture, or if you configure your go
installation to build to the desired architecture by default. See below for cross-compilation support.

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

Revel maintains a Heroku Buildpack, allowing one-command deploys:
https://github.com/robfig/heroku-buildpack-go-revel

Please see the
[README](https://github.com/robfig/heroku-buildpack-go-revel/blob/master/README.md)
for usage instructions.

## Cross-compilation

In order to create a cross-compile environment, we need to build go from source.
See
[Installing Go from source](http://golang.org/doc/install/source)
for more information.
You must properly set your $PATH and $GOPATH variables, otherwise if there is an existing
binary Go package, you will get into serious errors.

When you have a go compiler successfully setup, build the cross-compiler by
specifying the target environment with GOOS and GOARCH environment variables. See
[Optional environment variables](http://golang.org/doc/install/source#environment)
for more information.

    $ cd /path/to/goroot/src
    $ GOOS=linux GOARCH=amd64 ./make.bash --no-clean
    $ GOOS=windows GOARCH=386 ./make.bash --no-clean

Install revel on the new environment and you are set to go with the packaging.

    $ GOOS=linux GOARCH=amd64 revel package import/path/to/app

Copy the resulting tarball to your target platform.

