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

### Application settings

#### app.name

The human-readable application name. This is used for some console output and
development web pages.

Example:

    app.name = Booking example application

Default: no value

***
#### app.secret

The secret key used for cryptographic operations
([`revel.Sign`](../docs/godoc/util.html#Sign)).  Revel also uses it internally
to sign session cookies.  Setting it to empty string disables signing.

It is set to a random string when initializing a new project (using `revel new`)

Example:

	app.secret = pJLzyoiDe17L36mytqC912j81PfTiolHm1veQK6Grn1En3YFdB5lvEHVTwFEaWvj

Default: no value

### HTTP settings

#### http.port

The port to listen on.

Example:

	http.port = 9000

***
#### http.addr

The IP address on which to listen.

On Linux, empty string indicates a wildcard -- on Windows, empty string is
silently converted to `"localhost"`

Default: ""

***
#### harness.port

Specifies the port for the application to listen on, when run by the harness.
For example, when the harness is running, it will listen on `http.port`, run the
application on `harness.port`, and reverse-proxy requests.  Without the harness,
the application listens on `http.port` directly.

By default, a random free port will be chosen.  This is only necessary to set
when running in an environment that restricts socket access by the program.

Default: 0

***
#### http.ssl

If true, Revel's web server will configure itself to accept SSL connections. This
requires an X509 certificate and a key file.

Default: false

#### http.sslcert

Specifies the path to an X509 certificate file.

Default: ""

#### http.sslkey

Specifies the path to an X509 certificate key.

Default: ""

### Results

#### results.chunked

Determines whether the template rendering should use
[chunked encoding](en.wikipedia.org/wiki/Chunked_transfer_encoding).  Chunked
encoding can decrease the time to first byte on the client side by sending data
before the entire template has been fully rendered.

Default: false

***
#### results.pretty

Configures [`RenderXml`](../docs/godoc/controller.html#RenderXml) and
[`RenderJson`](../docs/godoc/controller.html#RenderJson) to produce indented
XML/JSON.  For example:

	results.pretty = true

Default: false

### Internationalization (i18n)

#### i18n.default_language

Specifies the default language for messages when the requested locale is not
recognized.  If left unspecified, a dummy message is returned to those requests.

For example:

	i18n.default_language = en

Default: ""

***
#### i18n.cookie

Specifies the name of the cookie used to store the user's locale.

Default: "%(cookie.prefix)_LANG" (see cookie.prefix)

### Watchers

Revel watches your project and supports hot-reload for a number of types of
source. To enable watching:

	watch = true

If false, nothing will be watched, regardless of the other `watch.*`
configuration keys.  (This is appropriate for production deployments)

Default: true

***
#### watch.templates

If true, Revel will watch your views for changes and reload them as necessary.

Default: true

***
#### watch.routes

If true, Revel will watch your `routes` file for changes and reload as
necessary.

Default: true

***
#### watch.code

If true, Revel will watch your Go code for changes and rebuild your application
as necessary.  (This runs the harness as a reverse-proxy to the application)

All code within the application's `app/` directory (or any sub-directory) is
watched.

Default: true

### Cookies

Revel components use the following cookies by default:
* REVEL_SESSION
* REVEL_LANG
* REVEL_FLASH
* REVEL_ERRORS

#### cookie.prefix

Revel uses this property as the prefix for the Revel-produced cookies. This is
so that multiple REVEL applications can coexist on the same host.

For example,

	cookie.prefix = MY

would result in the following cookie names:
* MY_SESSION
* MY_LANG
* MY_FLASH
* MY_ERRORS


Default: "REVEL"

### Session

#### session.expires

Revel uses this property to set the expiration of the session cookie.
Revel uses [ParseDuration](http://golang.org/pkg/time/#ParseDuration) to parse the string.
The default value is 30 days. It can also be set to "session" to allow session only
expiry. Please note that the client behaviour is dependent on browser configuration so
the result is not always guaranteed.

### Templates

#### template.delimiters 

Specifies an override for the left and right delimiters used in the templates.  
The delimiters must be specified as "LEFT\_DELIMS RIGHT\_DELIMS"

Default: "\{\{ \}\}"

### Formatting

#### format.date

Specifies the default date format for the application.  Revel uses this in two places:
* Binding dates to a `time.Time` (see [binding](binding.html))
* Printing dates using the `date` template function (see [template funcs](templates.html))

Default: "2006-01-02"

***
#### format.datetime

Specifies the default datetime format for the application.  Revel uses this in two places:
* Binding dates to a `time.Time` (see [binding](binding.html))
* Printing dates using the `datetime` template function (see [template funcs](templates.html))

Default: "2006-01-02 15:04"

### Database

#### db.import

Specifies the import path of the desired database/sql driver for the db module.

Default: ""

***
#### db.driver

Specifies the name of the database/sql driver (used in
[`sql.Open`](http://golang.org/pkg/database/sql/#Open)).

Default: ""

***
#### db.spec

Specifies the data source name of your database/sql database (used in
[`sql.Open`](http://golang.org/pkg/database/sql/#Open)).

Default: ""

### Build

#### build.tags

[Build tags](http://golang.org/cmd/go/#Compile_packages_and_dependencies) to use
when building an application.

Default: ""

### Logging

TODO

### Cache

The [cache](cache.html) module is a simple interface to a heap or distributed cache.

#### cache.expires

Sets the default duration before cache entries are expired from the cache.  It
is used when the caller passes the constant `cache.DEFAULT`.

It is specified as a duration string acceptable to
[`time.ParseDuration`](http://golang.org/pkg/time/#ParseDuration)

(Presently it is not possible to specify a default of `FOREVER`)

Default: "1h" (1 hour)

***
#### cache.memcached

If true, the cache module uses [memcached](http://memcached.org) instead of the
in-memory cache.

Default: false

***
#### cache.hosts

A comma-separated list of memcached hosts.  Cache entries are automatically
sharded among available hosts using a deterministic mapping of cache key to host
name.  Hosts may be listed multiple times to increase their share of cache
space.

Default: ""

### Scheduled Jobs

The [jobs](jobs.html) module allows you to run scheduled or ad-hoc jobs.

#### Named schedules

Named cron schedules may be configured by setting a key of the form:

	cron.schedulename = @hourly

That schedule may then be referenced upon submission to the job runner. For
example:

<pre class="prettyprint lang-go">
jobs.Schedule("cron.schedulename", job)
</pre>

***
#### jobs.pool

The number of jobs allowed to run concurrently.  For example:

	jobs.pool = 4

If 0, there is no limit imposed.

Default: 10

***
#### jobs.selfconcurrent

If true, allows a job to run even if previous instances of that job are still in
progress.

Default: false

### Modules

[Modules](modules.html) may be added to an application by specifying their base
import path.  For example:

	module.testrunner = github.com/robfig/revel/modules/testrunner

## Areas for development

* Allow inserting command line arguments as config values or otherwise
  specifying config values from the command line.
