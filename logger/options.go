package logger

import (
	"io"
	"os"
)

// Used for the callback to LogFunctionMap
type LogOptions struct {
	Ctx                    *config.Context
	ReplaceExistingHandler bool
	HandlerWrap            LogHandler
	Levels                 []LogLevel
	ExtendedOptions        map[string]interface{}
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

type Builder struct {
	Debug    []zap.Config
	Info     []zap.Config
	Warn     []zap.Config
	Error    []zap.Config
	Critical []zap.Config
}

func NewBuilder() *Builder {
	return &Builder{}
}
func (h *Builder) SetHandler(handler zap.Config, replace bool, level LogLevel) {
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
	case LvlCrit:
		source = &h.Critical
	}

	if !replace && *source != nil {
		*source = append(*source, handler)
	} else {
		*source = []zap.Config{handler}
	}
}

func (h *Builder) SetHandlers(handler LogHandler, options *LogOptions) {
	if len(options.Levels) == 0 {
		options.Levels = LvlAllList
	}
	// Set all levels
	for _, lvl := range options.Levels {
		h.SetHandler(handler, options.ReplaceExistingHandler, lvl)
	}
}

//func (h *Builder) SetJson(writer io.Writer, options *LogOptions) {
//cfg := zap.NewProductionConfig()
//cfg.Encoding  = "json"

//handler := CallerFileHandler(StreamHandler(writer, log15.JsonFormatEx(
//	options.GetBoolDefault("pretty", false),
//	options.GetBoolDefault("lineSeparated", true),
//)))
//if options.HandlerWrap != nil {
//	handler = options.HandlerWrap.SetChild(handler)
//}
//h.SetHandlers(handler, options)
//}

// Use built in rolling function
func (h *Builder) SetJsonFile(filePath string, options *LogOptions) {

	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"

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
	streamHandler := StreamHandler(
		writer,
		TerminalFormatHandler(
			options.GetBoolDefault("noColor", false),
			options.GetBoolDefault("smallDate", true)))

	if os.Stdout == writer {
		streamHandler = StreamHandler(
			colorable.NewColorableStdout(),
			TerminalFormatHandler(
				options.GetBoolDefault("noColor", false),
				options.GetBoolDefault("smallDate", true)))
	} else if os.Stderr == writer {
		streamHandler = StreamHandler(
			colorable.NewColorableStderr(),
			TerminalFormatHandler(
				options.GetBoolDefault("noColor", false),
				options.GetBoolDefault("smallDate", true)))
	}

	handler := CallerFileHandler(streamHandler)
	if options.HandlerWrap != nil {
		handler = options.HandlerWrap.SetChild(handler)
	}
	h.SetHandlers(handler, options)
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
