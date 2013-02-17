---
title: app.conf
layout: manual
---

## Overview

The application config file is named `app.conf` and uses the syntax accepted by
[goconfig](https://github.com/robfig/config), which is similar to Microsoft
INI files.

Here's an example file:

	app.name=chat
	app.secret=pJLzyoiDe17L36mytqC912j81PfTiolHm1veQK6Grn1En3YFdB5lvEHVTwFEaWvj
	http.addr=
	http.port=9000

	[dev]
	results.pretty=true
	watch=true

	log.trace.output = off
	log.info.output  = stderr
	log.warn.output  = stderr
	log.error.output = stderr

	[prod]
	results.pretty=false
	watch=false

	log.trace.output = off
	log.info.output  = off
	log.warn.output  = %(app.name)s.log
	log.error.output = %(app.name)s.log

Each section is a **Run Mode**.  The keys at the top level (not within any
section) apply to all run modes.  The key under the `[prod]` section applies
only to `prod` mode.  This allows default values to be supplied that apply
across all modes, and overridden as required.

New apps start with **dev** and **prod** run modes defined, but the user may
create any sections they wish.  The run mode is chosen at runtime by the
argument provided to "revel run" (the [command-line tool](tool.html)).

## Custom properties

The developer may define custom keys and access them via the
[`revel.Config` variable](../docs/godoc/revel.html#variables), which exposes a
[simple api](../docs/godoc/config.html).

## Built-in properties

Revel uses the following properties internally:
* app.name
* app.secret - the secret key used to sign session cookies (and anywhere the
  application uses [`revel.Sign`](../docs/godoc/util.html#Sign))
* http.port - the port to listen on
* http.addr - the ip address to which to bind (empty string is wildcard)
* results.pretty - [`RenderXml`](../docs/godoc/controller.html#RenderXml) and
  [`RenderJson`](../docs/godoc/controller.html#RenderJson) product nicely formatted
  XML/JSON.
* watch - enable source watching.  if false, no watching is done regardless of other watch settings.  (default True)
* watch.templates - should Revel watch for changes to views and reload?  (default True)
* watch.routes - should Revel watch for changes to routes and reload?  (default True)
* watch.code - should Revel watch for changes to code and reload?  (default True)
* cookie.prefix - how should the Revel-produced cookies be named?  (default "REVEL")
* log.* - Logging configuration

### Jobs

* cron.* - a named cron schedule
* jobs.pool - number of jobs allowed to run concurrently
* jobs.selfconcurrent - if true, allows a job to run even if previous instances have not completed yet.


## Areas for development

* Finish documenting the built-in properties
* Allow inserting command line arguments as config values or otherwise
  specifying config values from the command line.
