package logger

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/revel/config"
	"go.uber.org/zap"
)

// Utility package to make existing logging backwards compatible
var (
	// Convert the string to LogLevel
	toLevel = map[string]LogLevel{"debug": LvlDebug,
		"info":    LvlInfo,
		"request": LvlInfo,
		"warn":    LvlWarn,
		"error":   LvlError,
		"crit":    LvlCrit,
		"trace":   LvlTrace, // TODO trace is deprecated, replaced by debug
	}
)

func GetLogger(name string, logger MultiLogger) (l *log.Logger) {
	switch name {
	case "trace": // TODO trace is deprecated, replaced by debug
		l = logger.ToStdLogger(zap.DebugLevel)
	case "debug":
		l = logger.ToStdLogger(zap.DebugLevel)
	case "info":
		l = logger.ToStdLogger(zap.InfoLevel)
	case "warn":
		l = logger.ToStdLogger(zap.WarnLevel)
	case "error":
		l = logger.ToStdLogger(zap.ErrorLevel)
	case "request":
		l = logger.ToStdLogger(zap.InfoLevel)
	}
	return l
}

// Get all handlers based on the Config (if available)
func InitializeFromConfig(basePath string, config *config.Context) (c *Builder) {
	// If the configuration has an all option we can skip some
	c = NewBuilder()

	// Filters are assigned first, non filtered items override filters
	initAllLog(c, basePath, config)
	initLogLevels(c, basePath, config)
	if len(c.Critical) == 0 && len(c.Error) != 0 {
		c.Critical = c.Error
	}
	initFilterLog(c, basePath, config)
	if len(c.Critical) == 0 && len(c.Error) != 0 {
		c.Critical = c.Error
	}
	initRequestLog(c, basePath, config)

	return c
}

// Init the log.all configuration options
func initAllLog(c *Builder, basePath string, config *config.Context) {
	if config == nil {
		return
	}
	if output, found := config.String("log.all.output"); found {
		// Set all output for the specified handler
		log.Printf("Adding standard handler for levels to >%s< ", output)
		initHandlerFor(c, output, basePath, NewLogOptions(config, true, nil, LvlAllList...))
	}
}

// Init the filter options
// log.all.filter ....
// log.error.filter ....
func initFilterLog(c *Builder, basePath string, config *config.Context) {
	if config == nil {
		return
	}

	// The commands to use
	logFilterList := []struct {
		LogPrefix, LogSuffix string
		parentHandler        func(map[string]interface{}) ParentLogHandler
	}{{
		"log.", ".filter",
		func(keyMap map[string]interface{}) ParentLogHandler {
			return NewParentLogHandler(func(child LogHandler) LogHandler {
				return MatchMapHandler(keyMap, child)
			})

		},
	}, {
		"log.", ".nfilter",
		func(keyMap map[string]interface{}) ParentLogHandler {
			return NewParentLogHandler(func(child LogHandler) LogHandler {
				return NotMatchMapHandler(keyMap, child)
			})
		},
	}}

	for _, logFilter := range logFilterList {
		// Init for all filters
		for _, name := range []string{"all", "debug", "info", "warn", "error", "crit",
			"trace", // TODO trace is deprecated
		} {
			optionList := config.Options(logFilter.LogPrefix + name + logFilter.LogSuffix)
			for _, option := range optionList {
				splitOptions := strings.Split(option, ".")
				keyMap := map[string]interface{}{}
				for x := 3; x < len(splitOptions); x += 2 {
					keyMap[splitOptions[x]] = splitOptions[x+1]
				}
				phandler := logFilter.parentHandler(keyMap)
				log.Printf("Adding key map handler %s %s output %s", option, name, config.StringDefault(option, ""))

				if name == "all" {
					initHandlerFor(c, config.StringDefault(option, ""), basePath, NewLogOptions(config, false, phandler))
				} else {
					initHandlerFor(c, config.StringDefault(option, ""), basePath, NewLogOptions(config, false, phandler, toLevel[name]))
				}
			}
		}
	}
}

// Init the log.error, log.warn etc configuration options
func initLogLevels(c *Builder, basePath string, config *config.Context) {
	for _, name := range []string{"debug", "info", "warn", "error", "crit",
		"trace", // TODO trace is deprecated
	} {
		if config != nil {
			output, found := config.String("log." + name + ".output")
			if found {
				log.Printf("Adding standard handler %s output %s", name, output)
				initHandlerFor(c, output, basePath, NewLogOptions(config, true, nil, toLevel[name]))
			}
			// Gets the list of options with said prefix
		} else {
			initHandlerFor(c, "stderr", basePath, NewLogOptions(config, true, nil, toLevel[name]))
		}
	}
}

// Init the request log options
func initRequestLog(c *Builder, basePath string, config *config.Context) {
	// Request logging to a separate output handler
	// This takes the InfoHandlers and adds a MatchAbHandler handler to it to direct
	// context with the word "section=requestlog" to that handler.
	// Note if request logging is not enabled the MatchAbHandler will not be added and the
	// request log messages will be sent out the INFO handler
	outputRequest := "stdout"
	if config != nil {
		outputRequest = config.StringDefault("log.request.output", "")
	}
	oldInfo := c.InfoHandler
	c.InfoHandler = nil
	if outputRequest != "" {
		initHandlerFor(c, outputRequest, basePath, NewLogOptions(config, false, nil, LvlInfo))
	}
	if c.InfoHandler != nil || oldInfo != nil {
		if c.InfoHandler == nil {
			c.InfoHandler = oldInfo
		} else {
			c.InfoHandler = MatchAbHandler("section", "requestlog", c.InfoHandler, oldInfo)
		}
	}
}

// The log function map can be added to, so that you can specify your own logging mechanism
var LogFunctionMap = map[string]func(*Builder, *LogOptions){
	// Do nothing - set the logger off
	"off": func(c *Builder, logOptions *LogOptions) {
		// Only drop the results if there is a parent handler defined
		if logOptions.HandlerWrap != nil {
			for _, l := range logOptions.Levels {
				c.SetHandler(logOptions.HandlerWrap.SetChild(NilHandler()), logOptions.ReplaceExistingHandler, l)
			}
		}
	},
	// Do nothing - set the logger off
	"": func(*Builder, *LogOptions) {},
	// Set the levels to stdout, replace existing
	"stdout": func(c *Builder, logOptions *LogOptions) {
		if logOptions.Ctx != nil {
			logOptions.SetExtendedOptions(
				"noColor", !logOptions.Ctx.BoolDefault("log.colorize", true),
				"smallDate", logOptions.Ctx.BoolDefault("log.smallDate", true))
		}

		c.SetTerminalFile("stdout", logOptions)
	},
	// Set the levels to stderr output to terminal
	"stderr": func(c *Builder, logOptions *LogOptions) {
		c.SetTerminalFile("stderr", logOptions)
	},
}

func defaultInitHandlerFor(output string, c *Builder, logOptions *LogOptions) {
	// Write to file specified
	if !filepath.IsAbs(output) {
		output = filepath.Join(basePath, output)
	}

	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		log.Panic(err)
	}

	if strings.HasSuffix(output, "json") {
		c.SetJsonFile(output, options)
	} else {
		// Override defaults for a terminal file
		options.SetExtendedOptions("noColor", true)
		options.SetExtendedOptions("smallDate", false)
		c.SetTerminalFile(output, options)
	}
}

// Returns a handler for the level using the output string
// Accept formats for output string are
// LogFunctionMap[value] callback function
// `stdout` `stderr` `full/file/path/to/location/app.log` `full/file/path/to/location/app.json`
func initHandlerFor(c *Builder, output, basePath string, options *LogOptions) {
	if options.Ctx != nil {
		options.SetExtendedOptions(
			"noColor", !options.Ctx.BoolDefault("log.colorize", true),
			"smallDate", options.Ctx.BoolDefault("log.smallDate", true),
			"maxSize", options.Ctx.IntDefault("log.maxsize", 1024*10),
			"maxAge", options.Ctx.IntDefault("log.maxage", 14),
			"maxBackups", options.Ctx.IntDefault("log.maxbackups", 14),
			"compressBackups", !options.Ctx.BoolDefault("log.compressBackups", true),
		)
	}

	output = strings.TrimSpace(output)
	if funcHandler, found := LogFunctionMap[output]; found {
		funcHandler(c, options)
	} else {
		defaultHandler(output, c, options)
	}
}
