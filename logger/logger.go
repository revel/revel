package logger

import (
	"fmt"
	"github.com/revel/config"
	"time"
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
		Log(*Record) error
		//log15.Handler
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
		Format(r *Record) []byte
	}

	// The log level type
	LogLevel int

	// Used for the callback to LogFunctionMap
	LogOptions struct {
		Ctx                    *config.Context
		ReplaceExistingHandler bool
		HandlerWrap            ParentLogHandler
		Levels                 []LogLevel
		ExtendedOptions        map[string]interface{}
	}

	// The log record
	Record struct {
		Message string     // The message
		Time    time.Time  // The time
		Level   LogLevel   //The level
		Call    CallStack  // The call stack if built
		Context ContextMap // The context
	}

	// The lazy structure to implement a function to be invoked only if needed
	Lazy struct {
		Fn interface{} // the function
	}

	// Currently the only requirement for the callstack is to support the Formatter method
	// which stack.Call does so we use that
	CallStack interface {
		fmt.Formatter // Requirement
	}
)

// FormatFunc returns a new Format object which uses
// the given function to perform record formatting.
func FormatFunc(f func(*Record) []byte) LogFormat {
	return formatFunc(f)
}

type formatFunc func(*Record) []byte

func (f formatFunc) Format(r *Record) []byte {
	return f(r)
}
func NewRecord(message string, level LogLevel) *Record {
	return &Record{Message: message, Context: ContextMap{}, Level: level}
}

const (
	LvlCrit  LogLevel = iota // Critical
	LvlError                 // Error
	LvlWarn                  // Warning
	LvlInfo                  // Information
	LvlDebug                 // Debug
)

// A list of all the log levels
var LvlAllList = []LogLevel{LvlDebug, LvlInfo, LvlWarn, LvlError, LvlCrit}

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
