package logger

import (
	"github.com/mattn/go-colorable"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
)

type CompositeMultiHandler struct {
	DebugHandler    LogHandler
	InfoHandler     LogHandler
	WarnHandler     LogHandler
	ErrorHandler    LogHandler
	CriticalHandler LogHandler
}

func NewCompositeMultiHandler() (*CompositeMultiHandler, LogHandler) {
	cw := &CompositeMultiHandler{}
	return cw, cw
}
func (h *CompositeMultiHandler) Log(r *Record) (err error) {

	var handler LogHandler

	switch r.Level {
	case LvlInfo:
		handler = h.InfoHandler
	case LvlDebug:
		handler = h.DebugHandler
	case LvlWarn:
		handler = h.WarnHandler
	case LvlError:
		handler = h.ErrorHandler
	case LvlCrit:
		handler = h.CriticalHandler
	}

	// Embed the caller function in the context
	if handler != nil {
		handler.Log(r)
	}
	return
}

func (h *CompositeMultiHandler) SetHandler(handler LogHandler, replace bool, level LogLevel) {
	if handler == nil {
		// Ignore empty handler
		return
	}
	source := &h.DebugHandler
	switch level {
	case LvlDebug:
		source = &h.DebugHandler
	case LvlInfo:
		source = &h.InfoHandler
	case LvlWarn:
		source = &h.WarnHandler
	case LvlError:
		source = &h.ErrorHandler
	case LvlCrit:
		source = &h.CriticalHandler
	}

	if !replace && *source != nil {
		// If we are not replacing the source make sure that the level handler is applied first
		if _, isLevel := (*source).(*LevelFilterHandler); !isLevel {
			*source = LevelHandler(level, *source)
		}
		// If this already was a list add a new logger to it
		if ll, found := (*source).(*ListLogHandler); found {
			ll.Add(handler)
		} else {
			*source = NewListLogHandler(*source, handler)
		}
	} else {
		*source = handler
	}
}

// For the multi handler set the handler, using the LogOptions defined
func (h *CompositeMultiHandler) SetHandlers(handler LogHandler, options *LogOptions) {
	if len(options.Levels) == 0 {
		options.Levels = LvlAllList
	}

	// Set all levels
	for _, lvl := range options.Levels {
		h.SetHandler(handler, options.ReplaceExistingHandler, lvl)
	}

}
func (h *CompositeMultiHandler) SetJson(writer io.Writer, options *LogOptions) {
	handler := CallerFileHandler(StreamHandler(writer, JsonFormatEx(
		options.GetBoolDefault("pretty", false),
		options.GetBoolDefault("lineSeparated", true),
	)))
	if options.HandlerWrap != nil {
		handler = options.HandlerWrap.SetChild(handler)
	}
	h.SetHandlers(handler, options)
}

// Use built in rolling function
func (h *CompositeMultiHandler) SetJsonFile(filePath string, options *LogOptions) {
	writer := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    options.GetIntDefault("maxSizeMB", 1024), // megabytes
		MaxAge:     options.GetIntDefault("maxAgeDays", 7),   //days
		MaxBackups: options.GetIntDefault("maxBackups", 7),
		Compress:   options.GetBoolDefault("compress", true),
	}
	h.SetJson(writer, options)
}

func (h *CompositeMultiHandler) SetTerminal(writer io.Writer, options *LogOptions) {
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
func (h *CompositeMultiHandler) SetTerminalFile(filePath string, options *LogOptions) {
	writer := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    options.GetIntDefault("maxSizeMB", 1024), // megabytes
		MaxAge:     options.GetIntDefault("maxAgeDays", 7),   //days
		MaxBackups: options.GetIntDefault("maxBackups", 7),
		Compress:   options.GetBoolDefault("compress", true),
	}
	h.SetTerminal(writer, options)
}

func (h *CompositeMultiHandler) Disable(levels ...LogLevel) {
	if len(levels) == 0 {
		levels = LvlAllList
	}
	for _, level := range levels {
		switch level {
		case LvlDebug:
			h.DebugHandler = nil
		case LvlInfo:
			h.InfoHandler = nil
		case LvlWarn:
			h.WarnHandler = nil
		case LvlError:
			h.ErrorHandler = nil
		case LvlCrit:
			h.CriticalHandler = nil
		}
	}
}
