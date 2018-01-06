package logger

import (
	"io"

	"github.com/revel/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Used for the callback to LogFunctionMap
type LogOptions struct {
	Ctx                    *config.Context
	ReplaceExistingHandler bool
	// HandlerWrap            ParentHandler,
	Levels          []LogLevel
	ExtendedOptions map[string]interface{}
}

// Create a new log options
func NewLogOptions(cfg *config.Context, replaceHandler bool, lvl ...LogLevel) *LogOptions {
	return &LogOptions{
		Ctx: cfg,
		ReplaceExistingHandler: replaceHandler,
		// HandlerWrap:            phandler,
		Levels:          lvl,
		ExtendedOptions: map[string]interface{}{},
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

type Builder struct {
	Debug    []*zap.Logger
	Info     []*zap.Logger
	Warn     []*zap.Logger
	Error    []*zap.Logger
	Critical []*zap.Logger
}

func NewBuilder() *Builder {
	return &Builder{}
}
func (h *Builder) SetHandler(handler *zap.Logger, replace bool, level LogLevel) {
	if handler == nil {
		// Ignore empty handler
		return
	}

	source := &h.Debug
	switch level {
	case LvlDebug:
		source = &h.Debug
	case LvlInfo:
		source = &h.Info
	case LvlWarn:
		source = &h.Warn
	case LvlError:
		source = &h.Error
		//case LvlCrit:
		//	source = &h.Critical
	}

	if !replace && *source != nil {
		*source = append(*source, handler)
	} else {
		*source = []*zap.Logger{handler}
	}
}

func (h *Builder) SetHandlers(encoder zapcore.Encoder, writer io.Writer, options *LogOptions) {
	if len(options.Levels) == 0 {
		options.Levels = LvlAllList
	}
	// Set all levels
	for _, lvl := range options.Levels {
		core := zapcore.NewCore(
			encoder,
			zapcore.AddSync(writer),
			zap.InfoLevel,
		)
		h.SetHandler(zap.New(core), options.ReplaceExistingHandler, lvl)
	}
}

func (h *Builder) SetJson(writer io.Writer, options *LogOptions) {
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	h.SetHandlers(encoder, writer, options)
}

// Use built in rolling function
func (h *Builder) SetJsonFile(filePath string, options *LogOptions) {
	writer := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    options.GetIntDefault("maxSizeMB", 1024), // megabytes
		MaxAge:     options.GetIntDefault("maxAgeDays", 7),   //days
		MaxBackups: options.GetIntDefault("maxBackups", 7),
		Compress:   options.GetBoolDefault("compress", true),
	}
	h.SetJson(writer, options)
}

func (h *Builder) SetTerminal(writer io.Writer, options *LogOptions) {
	encoder := zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig())
	h.SetHandlers(encoder, writer, options)
}

// Use built in rolling function
func (h *Builder) SetTerminalFile(filePath string, options *LogOptions) {
	writer := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    options.GetIntDefault("maxSizeMB", 1024), // megabytes
		MaxAge:     options.GetIntDefault("maxAgeDays", 7),   //days
		MaxBackups: options.GetIntDefault("maxBackups", 7),
		Compress:   options.GetBoolDefault("compress", true),
	}
	h.SetTerminal(writer, options)
}

func (h *Builder) Disable(levels ...LogLevel) {
	if len(levels) == 0 {
		levels = LvlAllList
	}
	for _, level := range levels {
		switch level {
		case LvlDebug:
			h.Debug = nil
		case LvlInfo:
			h.Info = nil
		case LvlWarn:
			h.Warn = nil
		case LvlError:
			h.Error = nil
		case LvlCrit:
			h.Critical = nil
		}
	}
}
