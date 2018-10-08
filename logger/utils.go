package logger

import (
	"github.com/revel/log15"
	"gopkg.in/stack.v0"
	"log"
)

// Utility package to make existing logging backwards compatible
var (
	// Convert the string to LogLevel
	toLevel = map[string]LogLevel{"debug": LogLevel(log15.LvlDebug),
		"info": LogLevel(log15.LvlInfo), "request": LogLevel(log15.LvlInfo), "warn": LogLevel(log15.LvlWarn),
		"error": LogLevel(log15.LvlError), "crit": LogLevel(log15.LvlCrit),
		"trace": LogLevel(log15.LvlDebug), // TODO trace is deprecated, replaced by debug
	}
)

const (
	// The test mode flag overrides the default log level and shows only errors
	TEST_MODE_FLAG = "testModeFlag"
	// The special use flag enables showing messages when the logger is setup
	SPECIAL_USE_FLAG = "specialUseFlag"
)

// Returns the logger for the name
func GetLogger(name string, logger MultiLogger) (l *log.Logger) {
	switch name {
	case "trace": // TODO trace is deprecated, replaced by debug
		l = log.New(loggerRewrite{Logger: logger, Level: log15.LvlDebug}, "", 0)
	case "debug":
		l = log.New(loggerRewrite{Logger: logger, Level: log15.LvlDebug}, "", 0)
	case "info":
		l = log.New(loggerRewrite{Logger: logger, Level: log15.LvlInfo}, "", 0)
	case "warn":
		l = log.New(loggerRewrite{Logger: logger, Level: log15.LvlWarn}, "", 0)
	case "error":
		l = log.New(loggerRewrite{Logger: logger, Level: log15.LvlError}, "", 0)
	case "request":
		l = log.New(loggerRewrite{Logger: logger, Level: log15.LvlInfo}, "", 0)
	}

	return l

}

// Used by the initFilterLog to handle the filters
var logFilterList = []struct {
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

// This structure and method will handle the old output format and log it to the new format
type loggerRewrite struct {
	Logger         MultiLogger
	Level          log15.Lvl
	hideDeprecated bool
}

// The message indicating that a logger is using a deprecated log mechanism
var log_deprecated = []byte("* LOG DEPRECATED * ")

// Implements the Write of the logger
func (lr loggerRewrite) Write(p []byte) (n int, err error) {
	if !lr.hideDeprecated {
		p = append(log_deprecated, p...)
	}
	n = len(p)
	if len(p) > 0 && p[n-1] == '\n' {
		p = p[:n-1]
		n--
	}

	switch lr.Level {
	case log15.LvlInfo:
		lr.Logger.Info(string(p))
	case log15.LvlDebug:
		lr.Logger.Debug(string(p))
	case log15.LvlWarn:
		lr.Logger.Warn(string(p))
	case log15.LvlError:
		lr.Logger.Error(string(p))
	case log15.LvlCrit:
		lr.Logger.Crit(string(p))
	}

	return
}

// For logging purposes the call stack can be used to record the stack trace of a bad error
// simply pass it as a context field in your log statement like
// `controller.Log.Crit("This should not occur","stack",revel.NewCallStack())`
func NewCallStack() interface{} {
	return stack.Trace()
}
