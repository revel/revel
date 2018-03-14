package logger

import (
	"fmt"
	"github.com/revel/config"
	"github.com/revel/log15"
	"log"
	"os"
)

// The LogHandler defines the interface to handle the log records
type (
	// The Multilogger reduces the number of exposed defined logging variables,
	// and allows the output to be easily refined
	MultiLogger interface {
		// New returns a new Logger that has this logger's context plus the given context
		New(ctx ...interface{}) MultiLogger

		// SetHandler updates the logger to write records to the specified handler.
		SetHandler(h LogHandler)
		// Set the stack depth for the logger
		SetStackDepth(int) MultiLogger

		// Log a message at the given level with context key/value pairs
		Debug(msg string, ctx ...interface{})

		// Log a message at the given level formatting message with the parameters
		Debugf(msg string, params ...interface{})

		// Log a message at the given level with context key/value pairs
		Info(msg string, ctx ...interface{})

		// Log a message at the given level formatting message with the parameters
		Infof(msg string, params ...interface{})

		// Log a message at the given level with context key/value pairs
		Warn(msg string, ctx ...interface{})

		// Log a message at the given level formatting message with the parameters
		Warnf(msg string, params ...interface{})

		// Log a message at the given level with context key/value pairs
		Error(msg string, ctx ...interface{})

		// Log a message at the given level formatting message with the parameters
		Errorf(msg string, params ...interface{})

		// Log a message at the given level with context key/value pairs
		Crit(msg string, ctx ...interface{})

		// Log a message at the given level formatting message with the parameters
		Critf(msg string, params ...interface{})

		// Log a message at the given level with context key/value pairs and exits
		Fatal(msg string, ctx ...interface{})

		// Log a message at the given level formatting message with the parameters and exits
		Fatalf(msg string, params ...interface{})

		// Log a message at the given level with context key/value pairs and panics
		Panic(msg string, ctx ...interface{})

		// Log a message at the given level formatting message with the parameters and panics
		Panicf(msg string, params ...interface{})
	}

	// The log handler interface
	LogHandler interface {
		log15.Handler
	}

	// The log stack handler interface
	LogStackHandler interface {
		LogHandler
		GetStack() int
	}

	// The log handler interface which has child logs
	ParentLogHandler interface {
		SetChild(handler LogHandler) LogHandler
	}

	// The log format interface
	LogFormat interface {
		log15.Format
	}

	// The log level type
	LogLevel    log15.Lvl

	// This type implements the MultiLogger
	RevelLogger struct {
		log15.Logger
	}

	// Used for the callback to LogFunctionMap
	LogOptions struct {
		Ctx                    *config.Context
		ReplaceExistingHandler bool
		HandlerWrap            ParentLogHandler
		Levels                 []LogLevel
		ExtendedOptions        map[string]interface{}
	}
)

const (
	// Debug level
	LvlDebug = LogLevel(log15.LvlDebug)

	// Info level
	LvlInfo  = LogLevel(log15.LvlInfo)

	// Warn level
	LvlWarn  = LogLevel(log15.LvlWarn)

	// Error level
	LvlError = LogLevel(log15.LvlError)

	// Critical level
	LvlCrit  = LogLevel(log15.LvlCrit)
)

// A list of all the log levels
var LvlAllList = []LogLevel{LvlDebug, LvlInfo, LvlWarn, LvlError, LvlCrit}

// The log function map can be added to, so that you can specify your own logging mechanism
var LogFunctionMap = map[string]func(*CompositeMultiHandler, *LogOptions){
	// Do nothing - set the logger off
	"off": func(c *CompositeMultiHandler, logOptions *LogOptions) {
		// Only drop the results if there is a parent handler defined
		if logOptions.HandlerWrap != nil {
			for _, l := range logOptions.Levels {
				c.SetHandler(logOptions.HandlerWrap.SetChild(NilHandler()), logOptions.ReplaceExistingHandler, l)
			}
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

// Set the systems default logger
// Default logs will be captured and handled by revel at level info
func SetDefaultLog(fromLog MultiLogger) {
	log.SetOutput(loggerRewrite{Logger: fromLog, Level: log15.LvlInfo, hideDeprecated: true})
	// No need to show date and time, that will be logged with revel
	log.SetFlags(0)
}

// Print a formatted debug message
func (rl *RevelLogger) Debugf(msg string, param ...interface{}) {
	rl.Debug(fmt.Sprintf(msg, param...))
}

// Print a formatted info message
func (rl *RevelLogger) Infof(msg string, param ...interface{}) {
	rl.Info(fmt.Sprintf(msg, param...))
}

// Print a formatted warn message
func (rl *RevelLogger) Warnf(msg string, param ...interface{}) {
	rl.Warn(fmt.Sprintf(msg, param...))
}

// Print a formatted error message
func (rl *RevelLogger) Errorf(msg string, param ...interface{}) {
	rl.Error(fmt.Sprintf(msg, param...))
}

// Print a formatted critical message
func (rl *RevelLogger) Critf(msg string, param ...interface{}) {
	rl.Crit(fmt.Sprintf(msg, param...))
}

// Print a formatted fatal message
func (rl *RevelLogger) Fatalf(msg string, param ...interface{}) {
	rl.Fatal(fmt.Sprintf(msg, param...))
}

// Print a formatted panic message
func (rl *RevelLogger) Panicf(msg string, param ...interface{}) {
	rl.Panic(fmt.Sprintf(msg, param...))
}

// Print a critical message and call os.Exit(1)
func (rl *RevelLogger) Fatal(msg string, ctx ...interface{}) {
	rl.Crit(msg, ctx...)
	os.Exit(1)
}

// Print a critical message and panic
func (rl *RevelLogger) Panic(msg string, ctx ...interface{}) {
	rl.Crit(msg, ctx...)
	panic(msg)
}

// Override log15 method
func (rl *RevelLogger) New(ctx ...interface{}) MultiLogger {
	old := &RevelLogger{Logger: rl.Logger.New(ctx...)}
	return old
}

// Set the stack level to check for the caller
func (rl *RevelLogger) SetStackDepth(amount int) MultiLogger {
	rl.Logger.SetStackDepth(amount) // Ignore the logger returned
	return rl
}

// Create a new logger
func New(ctx ...interface{}) MultiLogger {
	r := &RevelLogger{Logger: log15.New(ctx...)}
	r.SetStackDepth(1)
	return r
}

// Set the handler in the Logger
func (rl *RevelLogger) SetHandler(h LogHandler) {
	rl.Logger.SetHandler(h)
}

// Implements the ParentLogHandler
type parentLogHandler struct {
	setChild func(handler LogHandler) LogHandler
}

// Create a new parent log handler
func NewParentLogHandler(callBack func(child LogHandler) LogHandler) ParentLogHandler {
	return &parentLogHandler{callBack}
}

// Sets the child of the log handler
func (p *parentLogHandler) SetChild(child LogHandler) LogHandler {
	return p.setChild(child)
}

// Create a new log options
func NewLogOptions(cfg *config.Context, replaceHandler bool, phandler ParentLogHandler, lvl ...LogLevel) (logOptions *LogOptions) {
	logOptions = &LogOptions{
		Ctx: cfg,
		ReplaceExistingHandler: replaceHandler,
		HandlerWrap:            phandler,
		Levels:                 lvl,
		ExtendedOptions:        map[string]interface{}{},
	}
	return
}

// Assumes options will be an even number and have a string, value syntax
func (l *LogOptions) SetExtendedOptions(options ...interface{}) {
	for x := 0; x < len(options); x += 2 {
		l.ExtendedOptions[options[x].(string)] = options[x+1]
	}
}

// Gets a string option with default
func (l *LogOptions) GetStringDefault(option, value string) string {
	if v, found := l.ExtendedOptions[option]; found {
		return v.(string)
	}
	return value
}

// Gets an int option with default
func (l *LogOptions) GetIntDefault(option string, value int) int {
	if v, found := l.ExtendedOptions[option]; found {
		return v.(int)
	}
	return value
}

// Gets a boolean option with default
func (l *LogOptions) GetBoolDefault(option string, value bool) bool {
	if v, found := l.ExtendedOptions[option]; found {
		return v.(bool)
	}
	return value
}
