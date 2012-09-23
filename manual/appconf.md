---
title: app.conf
layout: manual
---

## Overview

The application config file is named `app.conf` and uses the syntax accepted by
[goconfig](https://github.com/robfig/goconfig), which is similar to Microsoft
INI files.

Here's an example file:

	app.name=chat
	app.secret=pJLzyoiDe17L36mytqC912j81PfTiolHm1veQK6Grn1En3YFdB5lvEHVTwFEaWvj
	http.addr=
	http.port=9000

	[dev]
	results.pretty=true
	server.watcher=true

	log.trace.output = off
	log.info.output  = stderr
	log.warn.output  = stderr
	log.error.output = stderr

	[prod]
	results.pretty=false
	server.watcher=false

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
[`rev.Config` variable](../docs/godoc/revel.html#variables), which exposes a
[simple api](../docs/godoc/config.html).

## Built-in properties

Revel uses the following properties internally:
* app.name
* app.secret - the secret key used to sign session cookies (and anywhere the
  application uses [`rev.Sign`](../docs/godoc/util.html#Sign))
* http.port - the port to listen on
* http.addr - the ip address to which to bind (empty string is wildcard)
* results.pretty - [`RenderXml`](../docs/godoc/mvc.html#RenderXml) and
  [`RenderJson`](../docs/godoc/mvc.html#RenderJson) product nicely formatted
  XML/JSON.
* server.watcher - should Revel watch for changes to files and reload?
* log.* - Logging configuration
