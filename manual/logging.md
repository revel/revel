---
title: Logging
layout: manual
---

Revel provides four loggers:
* TRACE - debugging information only.
* INFO - informational.
* WARN - something unexpected but not harmful.
* ERROR - someone should take a look at this.

Example usage within a Revel app:

	now := time.Now()
	revel.TRACE.Printf("%s", now.String())

Each of these is a variable to a default [go logger](http://golang.org/pkg/log/).

Loggers may be configured in [app.conf](appconf.html).  Here is an example:

	app.name = sampleapp

	[dev]
	log.trace.output = stdout
	log.info.output  = stdout
	log.warn.output  = stderr
	log.error.output = stderr

	log.trace.prefix = "TRACE "
	log.info.prefix  = "INFO  "

	log.trace.flags  = 10
	log.info.flags   = 10

	[prod]
	log.trace.output = off
	log.info.output  = off
	log.warn.output  = log/%(app.name)s.log
	log.error.output = log/%(app.name)s.log


In **dev** mode:

* even the most detailed logs will be shown.
* everything logged at **info** or **trace** will be prefixed with its logging
level.

In **prod** mode:

* **info** and **trace** logs are ignored.
* both warnings and errors are appended to the **log/sampleapp.log** file.

To specify logger flags, you must calculate the flag value from
[the flag constants](http://www.golang.org/pkg/log/#constants).  For example, to
the format `01:23:23 /a/b/c/d.go:23 Message` requires the flags
`Ltime | Llongfile = 2 | 8 = 10`.

Areas for development:

* Revel should create the log directory if it does not already exist.
