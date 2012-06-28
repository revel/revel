---
title: app.conf
layout: manual
---

The application config file is named `app.conf` and uses the syntax accepted by
[goconfig](https://github.com/kless/goconfig), which is similar to Microsoft INI
files.

Here's an example file:

	app.name=chat
	app.mode=dev
	app.secret=pJLzyoiDe17L36mytqC912j81PfTiolHm1veQK6Grn1En3YFdB5lvEHVTwFEaWvj
	http.port=9000

	[prod]
	app.mode=prod


Revel treats the sections as "environments".  The keys at the top level (not
within any section) apply to all environments.  The key under the `[prod]`
section applies only to the `prod` environment.  This allows default values to
be supplied that apply across all environments, and overridden as required.

The environment is set on startup by the [command-line tool](tool.md), and it
determines the values read from the config by Revel and the application alike.

## Custom properties

The developer may define custom keys and access them via the
[`rev.Config` variable](../docs/godoc/revel.html#variables), which exposes a
[simple api](../docs/godoc/config.html).

## Built-in properties

Revel uses the following properties internally:
* app.name
* app.mode - determines the value of the `RunMode` variable.
* app.secret - the secret key used to sign session cookies (and anywhere the
  application uses [`rev.Sign`](../docs/godoc/util.html#Sign)
* http.port - the port to listen on

