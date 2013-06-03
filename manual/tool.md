---
title: Command-line Tool
layout: manual
---

## Build and Run

You must build the command line tool in order to use Revel:

	$ go get github.com/robfig/revel/revel

Now run it:

	$ bin/revel
	~
	~ revel! http://robfig.github.com/revel
	~
	usage: revel command [arguments]

	The commands are:

		new         create a skeleton Revel application
		run         run a Revel application
		build       build a Revel application (e.g. for deployment)
		package     package a Revel application (e.g. for deployment)
		clean       clean a Revel application's temp files
		test        run all tests from the command-line

	Use "revel help [command]" for more information.

Please refer to the tool's built-in help functionality for information on the
individual commands.
