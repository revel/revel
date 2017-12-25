package logger

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/revel/config"
	"github.com/revel/log15"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// The LogHandler defines the interface to handle the log records
type (
	// The Multilogger reduces the number of exposed defined logging variables,
	// and allows the output to be easily refined
	MultiLogger interface {
		//log15.Logger
		//// New returns a new Logger that has this logger's context plus the given context
		New(ctx ...interface{}) MultiLogger
		//
		//// SetHandler updates the logger to write records to the specified handler.
		SetHandler(h LogHandler)

		//
		//// Log a message at the given level with context key/value pairs
		Debug(msg string, ctx ...interface{})
		Debugf(msg string, params ...interface{})
		Info(msg string, ctx ...interface{})
		Infof(msg string, params ...interface{})
		Warn(msg string, ctx ...interface{})
		Warnf(msg string, params ...interface{})
		Error(msg string, ctx ...interface{})
		Errorf(msg string, params ...interface{})
		Crit(msg string, ctx ...interface{})
		Critf(msg string, params ...interface{})

		//// Logs a message as an Crit and exits
		Fatal(msg string, ctx ...interface{})
		Fatalf(msg string, params ...interface{})
		//// Logs a message as an Crit and panics
		Panic(msg string, ctx ...interface{})
		Panicf(msg string, params ...interface{})
	}

	LogLevel zapcore.Level

	RevelLogger struct {
		l *zap.Logger
		s *zap.SugaredLogger
	}

	// Used for the callback to LogFunctionMap
	LogOptions struct {
		Ctx                    *config.Context
		ReplaceExistingHandler bool
		HandlerWrap            LogHandler
		Levels                 []LogLevel
		ExtendedOptions        map[string]interface{}
	}
)

const (
	LvlDebug = zap.DebugLevel
	LvlInfo  = zap.InfoLevel
	LvlWarn  = zap.WarnLevel
	LvlError = zap.ErrorLevel
	LvlCrit  = zap.InfoLevel
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

func (rl *RevelLogger) Debugf(msg string, param ...interface{}) {
	rl.s.Debugf(msg, args...)
}
func (rl *RevelLogger) Infof(msg string, param ...interface{}) {
	rl.s.Infof(msg, args...)
}
func (rl *RevelLogger) Warnf(msg string, param ...interface{}) {
	rl.s.Warnf(msg, args...)
}
func (rl *RevelLogger) Errorf(msg string, param ...interface{}) {
	rl.s.Errorf(msg, args...)
}
func (rl *RevelLogger) Critf(msg string, param ...interface{}) {
	rl.s.Infof(msg, args...)
}
func (rl *RevelLogger) Fatalf(msg string, param ...interface{}) {
	rl.s.Fatalf(msg, args...)
}
func (rl *RevelLogger) Panicf(msg string, param ...interface{}) {
	rl.s.Panicf(msg, args...)
}

func (rl *RevelLogger) Fatal(msg string, ctx ...interface{}) {
	rl.s.Fatalw(msg, ctx...)
}

func (rl *RevelLogger) Panic(msg string, ctx ...interface{}) {
	rl.Crit(msg, ctx...)
	panic(msg)
}

func ctxToFields(ctx ...interface{}) []zapcore.Field {
	if len(ctx)%2 != 0 {
		panic(errors.New("ctx is invalid"))
	}
	fields := make([]zapcore.Field, 0, len(ctx)/2)
	for i := 0; i < len(ctx); i += 2 {
		fields = append(fields, zap.Any(fmt.Sprint(ctx[i]), ctx[i+1]))
	}
	return fields
}

// Override log15 method
func (rl *RevelLogger) New(ctx ...interface{}) MultiLogger {
	logger := rl.l.With(ctxToFields(ctx)...)
	return &RevelLogger{l: logger, s: logger.Sugar()}
}

// Set the stack level to check for the caller
func (rl *RevelLogger) SetStackDepth(amount int) MultiLogger {
	rl.Logger.SetStackDepth(amount) // Ignore the logger returned
	return rl
}

// Create a new logger
func New(ctx ...interface{}) MultiLogger {
	return zap.L().With(ctxToFields(ctx)).WithOptions(zap.AddCallerSkip(1))
}

// Set the handler in the Logger
func (rl *RevelLogger) SetHandler(h LogHandler) {
	rl.Logger.SetHandler(h)
}

// Create a new log options
func NewLogOptions(cfg *config.Context, replaceHandler bool, phandler ParentLogHandler, lvl ...LogLevel) *LogOptions {
	return &LogOptions{
		Ctx: cfg,
		ReplaceExistingHandler: replaceHandler,
		HandlerWrap:            phandler,
		Levels:                 lvl,
		ExtendedOptions:        map[string]interface{}{},
	}
}

// Assumes options will be an even number and have a string, value syntax
func (l *LogOptions) SetExtendedOptions(options ...interface{}) {
	for x := 0; x < len(options); x += 2 {
		l.ExtendedOptions[options[x].(string)] = options[x+1]
	}
}
func (l *LogOptions) GetStringDefault(option, value string) string {
	if v, found := l.ExtendedOptions[option]; found {
		return v.(string)
	}
	return value
}
func (l *LogOptions) GetIntDefault(option string, value int) int {
	if v, found := l.ExtendedOptions[option]; found {
		return v.(int)
	}
	return value
}
func (l *LogOptions) GetBoolDefault(option string, value bool) bool {
	if v, found := l.ExtendedOptions[option]; found {
		return v.(bool)
	}
	return value
}
