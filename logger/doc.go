/*
  Package logger contains filters and handles for the logging utilities in Revel.
  These facilities all currently use the logging library called log15 at
  https://github.com/inconshreveable/log15

	Defining handlers happens as follows
	1) ALL handlers (log.all.output) replace any existing handlers
	2) Output handlers (log.error.output) replace any existing handlers
	3) Filter handlers (log.xxx.filter, log.xxx.nfilter) append to existing handlers,
	   note log.all.filter is treated as a filter handler, so it will NOT replace existing ones



*/
package logger
