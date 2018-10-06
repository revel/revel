package logger

import (
	"os"
)

// The log function map can be added to, so that you can specify your own logging mechanism
// it has defaults for off, stdout, stderr
var LogFunctionMap = map[string]func(*CompositeMultiHandler, *LogOptions){
	// Do nothing - set the logger off
	"off": func(c *CompositeMultiHandler, logOptions *LogOptions) {
		// Only drop the results if there is a parent handler defined
		if logOptions.HandlerWrap != nil {
			for _, l := range logOptions.Levels {
				c.SetHandler(logOptions.HandlerWrap.SetChild(NilHandler()), logOptions.ReplaceExistingHandler, l)
			}
		} else {
			// Clear existing handler
			c.SetHandlers(NilHandler(), logOptions)
		}
	},
	// Do nothing - set the logger off
	"": func(*CompositeMultiHandler, *LogOptions) {},
	// Set the levels to stdout, replace existing
	"stdout": func(c *CompositeMultiHandler, logOptions *LogOptions) {
		if logOptions.Ctx != nil {
			logOptions.SetExtendedOptions(
				"noColor", !logOptions.Ctx.BoolDefault("log.colorize", true),
				"smallDate", logOptions.Ctx.BoolDefault("log.smallDate", true))
		}
		c.SetTerminal(os.Stdout, logOptions)
	},
	// Set the levels to stderr output to terminal
	"stderr": func(c *CompositeMultiHandler, logOptions *LogOptions) {
		c.SetTerminal(os.Stderr, logOptions)
	},
}
