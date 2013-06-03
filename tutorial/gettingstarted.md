---
title: Getting Started
layout: tutorial
---

This article walks through the installation process.

#### Install Go

To use Revel, you first need to [install Go](http://golang.org/doc/install).

#### Set up your GOPATH

If you did not create a GOPATH as part of installation, do so now.  Your GOPATH
is a directory tree where all of your Go code will live.  Here are the steps to do that:

1. Make a directory: `mkdir ~/gocode`
2. Tell Go to use that as your GOPATH: `export GOPATH=~/gocode`
3. Save your GOPATH so that it will apply to all future shell sessions: `echo GOPATH=$GOPATH >> .bash_profile`

Now your Go installation is complete.

#### Install git and hg

Both Git and Mercurial are required to allow `go get` to clone various dependencies.

* [Installing Git](http://git-scm.com/book/en/Getting-Started-Installing-Git)
* [Installing Mercurial](http://mercurial.selenic.com/wiki/Download)

#### Get the Revel framework

To get the Revel framework, run

	go get github.com/robfig/revel

This command does a couple things:

* Go uses git to clone the repository into `$GOPATH/src/github.com/robfig/revel/`
* Go transitively finds all of the dependencies and runs `go get` on them as well.

#### Build the Revel command line tool

The Revel command line tool is how you build, run, and package Revel applications.

Use `go get` to install it:

	go get github.com/robfig/revel/revel

Then, ensure the $GOPATH/bin directory is in your PATH so that you can reference the command from anywhere.

	export PATH="$PATH:$GOPATH/bin"
	echo 'PATH="$PATH:$GOPATH/bin"' >> .bash_profile

Lastly, let's verify that it works:

	$ revel help
	~
	~ revel! http://robfig.github.com/revel
	~
	usage: revel command [arguments]

	The commands are:

	    run         run a Revel application
	    new         create a skeleton Revel application
	    clean       clean a Revel application's temp files
	    package     package a Revel application (e.g. for deployment)

	Use "revel help [command]" for more information.

Now we are all set up.

**Next: [Create a new Revel application.](createapp.html)**
