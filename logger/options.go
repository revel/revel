package logger

import (
	"github.com/revel/config"
	"go.uber.org/zap/zapcore"
)

type HandlerWrapper func(core zapcore.Core, options *LogOptions) zapcore.Core

// Used for the callback to LogFunctionMap
type LogOptions struct {
	Ctx                    *config.Context
	ReplaceExistingHandler bool
	HandlerWrap            HandlerWrapper
	Levels                 []LogLevel
	ExtendedOptions        map[string]interface{}
}

// Create a new log options
func NewLogOptions(cfg *config.Context, replaceHandler bool, wrapper HandlerWrapper, lvl ...LogLevel) *LogOptions {
	return &LogOptions{
		Ctx: cfg,
		ReplaceExistingHandler: replaceHandler,
		HandlerWrap:            wrapper,
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
