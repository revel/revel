---
title: Command-line Tool
layout: manual
---

## Build and Run

You must build the command line tool in order to use Revel.  From the root of
your GOPATH:

	$ go build -o bin/revel github.com/robfig/revel/cmd

Now run it:

	$ bin/revel
	~
	~ revel! http://robfig.github.com/revel
	~
	usage: revel command [arguments]

	The commands are:

	    run         run a Revel application
	    new         create a skeleton Revel application
	    clean       clean a Revel application's temp files
	    package     package a Revel application (e.g. for deployment)
	    test        run all tests from the command-line

	Use "revel help [command]" for more information.

Please refer to the tool's built-in help functionality for information on the
individual commands.
