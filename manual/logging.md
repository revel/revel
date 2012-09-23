---
title: Logging
layout: manual
---

Revel provides four loggers:
* TRACE - debugging information only.
* INFO - informational.
* WARN - something unexpected but not harmful.
* ERROR - someone should take a look at this.

Loggers may be configured in [app.conf](appconf.html).  Here is an example:

	app.name = sampleapp

	[dev]
	log.trace.output = stdout
	log.info.output  = stdout
	log.warn.output  = stderr
	log.error.output = stderr

	log.trace.prefix = "TRACE "
	log.trace.prefix = "INFO  "

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

Areas for development:

* Revel should create the log directory if it does not already exist.
* Revel should have support for specifying flags.
