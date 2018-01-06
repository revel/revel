package logger

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// The LogHandler defines the interface to handle the log records
type (
	// The Multilogger reduces the number of exposed defined logging variables,
	// and allows the output to be easily refined
	MultiLogger interface {
		// New returns a new Logger that has this logger's context plus the given context
		New(ctx ...interface{}) MultiLogger

		// Log a message at the given level with context key/value pairs
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
)

const (
	LvlTrace = LogLevel(zap.DebugLevel)
	LvlDebug = LogLevel(zap.DebugLevel)
	LvlInfo  = LogLevel(zap.InfoLevel)
	LvlWarn  = LogLevel(zap.WarnLevel)
	LvlError = LogLevel(zap.ErrorLevel)
	LvlCrit  = LogLevel(zap.InfoLevel)
)

// A list of all the log levels
var LvlAllList = []LogLevel{LvlDebug, LvlInfo, LvlWarn, LvlError, LvlCrit}

func (rl *RevelLogger) Debugf(msg string, params ...interface{}) {
	rl.s.Debugf(msg, params...)
}
func (rl *RevelLogger) Infof(msg string, params ...interface{}) {
	rl.s.Infof(msg, params...)
}
func (rl *RevelLogger) Warnf(msg string, params ...interface{}) {
	rl.s.Warnf(msg, params...)
}
func (rl *RevelLogger) Errorf(msg string, params ...interface{}) {
	rl.s.Errorf(msg, params...)
}
func (rl *RevelLogger) Critf(msg string, params ...interface{}) {
	rl.s.Infof(msg, params...)
}
func (rl *RevelLogger) Fatalf(msg string, params ...interface{}) {
	rl.s.Fatalf(msg, params...)
}
func (rl *RevelLogger) Panicf(msg string, params ...interface{}) {
	rl.s.Panicf(msg, params...)
}

func (rl *RevelLogger) Debug(msg string, ctx ...interface{}) {
	rl.s.Debugw(msg, ctx...)
}
func (rl *RevelLogger) Info(msg string, ctx ...interface{}) {
	rl.s.Infow(msg, ctx...)
}
func (rl *RevelLogger) Warn(msg string, ctx ...interface{}) {
	rl.s.Warnw(msg, ctx...)
}
func (rl *RevelLogger) Error(msg string, ctx ...interface{}) {
	rl.s.Errorw(msg, ctx...)
}
func (rl *RevelLogger) Crit(msg string, ctx ...interface{}) {
	rl.s.Infow(msg, ctx...)
}
func (rl *RevelLogger) Fatal(msg string, ctx ...interface{}) {
	rl.s.Fatalw(msg, ctx...)
}
func (rl *RevelLogger) Panic(msg string, ctx ...interface{}) {
	rl.s.Panicw(msg, ctx...)
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

// Create a new logger
func New(ctx ...interface{}) MultiLogger {
	logger := zap.L().With(ctxToFields(ctx)...).WithOptions(zap.AddCallerSkip(1))
	return &RevelLogger{l: logger, s: logger.Sugar()}
}
