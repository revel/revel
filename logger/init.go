package logger

// Get all handlers based on the Config (if available)
import (
	"fmt"
	"github.com/revel/config"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func InitializeFromConfig(basePath string, config *config.Context) (c *CompositeMultiHandler) {
	// If running in test mode suppress anything that is not an error
	if config != nil && config.BoolDefault(TEST_MODE_FLAG, false) {
		// Preconfigure all the options
		config.SetOption("log.info.output", "none")
		config.SetOption("log.debug.output", "none")
		config.SetOption("log.warn.output", "none")
		config.SetOption("log.error.output", "stderr")
		config.SetOption("log.crit.output", "stderr")
	}

	// If the configuration has an all option we can skip some
	c, _ = NewCompositeMultiHandler()

	// Filters are assigned first, non filtered items override filters
	if config != nil && !config.BoolDefault(TEST_MODE_FLAG, false) {
		initAllLog(c, basePath, config)
	}
	initLogLevels(c, basePath, config)
	if c.CriticalHandler == nil && c.ErrorHandler != nil {
		c.CriticalHandler = c.ErrorHandler
	}
	if config != nil && !config.BoolDefault(TEST_MODE_FLAG, false) {
		initFilterLog(c, basePath, config)
		if c.CriticalHandler == nil && c.ErrorHandler != nil {
			c.CriticalHandler = c.ErrorHandler
		}
		initRequestLog(c, basePath, config)
	}

	return c
}

// Init the log.all configuration options
func initAllLog(c *CompositeMultiHandler, basePath string, config *config.Context) {
	if config != nil {
		extraLogFlag := config.BoolDefault(SPECIAL_USE_FLAG, false)
		if output, found := config.String("log.all.output"); found {
			// Set all output for the specified handler
			if extraLogFlag {
				log.Printf("Adding standard handler for levels to >%s< ", output)
			}
			initHandlerFor(c, output, basePath, NewLogOptions(config, true, nil, LvlAllList...))
		}
	}
}

// Init the filter options
// log.all.filter ....
// log.error.filter ....
func initFilterLog(c *CompositeMultiHandler, basePath string, config *config.Context) {

	if config != nil {
		extraLogFlag := config.BoolDefault(SPECIAL_USE_FLAG, false)

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
					if extraLogFlag {
						log.Printf("Adding key map handler %s %s output %s", option, name, config.StringDefault(option, ""))
						fmt.Printf("Adding key map handler %s %s output %s matching %#v\n", option, name, config.StringDefault(option, ""), keyMap)
					}

					if name == "all" {
						initHandlerFor(c, config.StringDefault(option, ""), basePath, NewLogOptions(config, false, phandler))
					} else {
						initHandlerFor(c, config.StringDefault(option, ""), basePath, NewLogOptions(config, false, phandler, toLevel[name]))
					}
				}
			}
		}
	}
}

// Init the log.error, log.warn etc configuration options
func initLogLevels(c *CompositeMultiHandler, basePath string, config *config.Context) {
	for _, name := range []string{"debug", "info", "warn", "error", "crit",
		"trace", // TODO trace is deprecated
	} {
		if config != nil {
			extraLogFlag := config.BoolDefault(SPECIAL_USE_FLAG, false)
			output, found := config.String("log." + name + ".output")
			if found {
				if extraLogFlag {
					log.Printf("Adding standard handler %s output %s", name, output)
				}
				initHandlerFor(c, output, basePath, NewLogOptions(config, true, nil, toLevel[name]))
			}
			// Gets the list of options with said prefix
		} else {
			initHandlerFor(c, "stderr", basePath, NewLogOptions(config, true, nil, toLevel[name]))
		}
	}
}

// Init the request log options
func initRequestLog(c *CompositeMultiHandler, basePath string, config *config.Context) {
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

// Returns a handler for the level using the output string
// Accept formats for output string are
// LogFunctionMap[value] callback function
// `stdout` `stderr` `full/file/path/to/location/app.log` `full/file/path/to/location/app.json`
func initHandlerFor(c *CompositeMultiHandler, output, basePath string, options *LogOptions) {
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
		switch output {
		case "":
			fallthrough
		case "off":
		// No handler, discard data
		default:
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
	}
	return
}
