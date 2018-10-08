package logger

import (
	"fmt"
	"github.com/revel/log15"
	"log"
	"os"
)

// This type implements the MultiLogger
type RevelLogger struct {
	log15.Logger
}

// Set the systems default logger
// Default logs will be captured and handled by revel at level info
func SetDefaultLog(fromLog MultiLogger) {
	log.SetOutput(loggerRewrite{Logger: fromLog, Level: log15.LvlInfo, hideDeprecated: true})
	// No need to show date and time, that will be logged with revel
	log.SetFlags(0)
}

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
	rl.Logger.SetHandler(callHandler(h.Log))
}

// The function wrapper to implement the callback
type callHandler func(r *Record) error

// Log implementation, reads the record and extracts the details from the log record
// Hiding the implementation.
func (c callHandler) Log(log *log15.Record) error {
	ctx := log.Ctx
	var ctxMap ContextMap
	if len(ctx) > 0 {
		ctxMap = make(ContextMap, len(ctx)/2)

		for i := 0; i < len(ctx); i += 2 {
			v := ctx[i]
			key, ok := v.(string)
			if !ok {
				key = fmt.Sprintf("LOGGER_INVALID_KEY %v", v)
			}
			var value interface{}
			if len(ctx) > i+1 {
				value = ctx[i+1]
			} else {
				value = "LOGGER_VALUE_MISSING"
			}
			ctxMap[key] = value
		}
	} else {
		ctxMap = make(ContextMap, 0)
	}
	r := &Record{Message: log.Msg, Context: ctxMap, Time: log.Time, Level: LogLevel(log.Lvl), Call: CallStack(log.Call)}
	return c(r)
}

// Internally used contextMap, allows conversion of map to map[string]string
type ContextMap map[string]interface{}

// Convert the context map to be string only values, any non string values are ignored
func (m ContextMap) StringMap() (newMap map[string]string) {
	if m != nil {
		newMap = map[string]string{}
		for key, value := range m {
			if svalue, isstring := value.(string); isstring {
				newMap[key] = svalue
			}
		}
	}
	return
}
func (m ContextMap) Add(key string, value interface{}) {
	m[key] = value
}
