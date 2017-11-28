package logger

import (
	"github.com/revel/log15"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
)

// Filters out records which do not match the level
// Uses the `log15.FilterHandler` to perform this task
func LevelHandler(lvl LogLevel, h LogHandler) LogHandler {
	l15Lvl := log15.Lvl(lvl)
	return log15.FilterHandler(func(r *log15.Record) (pass bool) {
		return r.Lvl == l15Lvl
	}, h)
}

// Filters out records which do not match the level
// Uses the `log15.FilterHandler` to perform this task
func MinLevelHandler(lvl LogLevel, h LogHandler) LogHandler {
	l15Lvl := log15.Lvl(lvl)
	return log15.FilterHandler(func(r *log15.Record) (pass bool) {
		return r.Lvl <= l15Lvl
	}, h)
}

// Filters out records which match the level
// Uses the `log15.FilterHandler` to perform this task
func NotLevelHandler(lvl LogLevel, h LogHandler) LogHandler {
	l15Lvl := log15.Lvl(lvl)
	return log15.FilterHandler(func(r *log15.Record) (pass bool) {
		return r.Lvl != l15Lvl
	}, h)
}

// Adds in a context called `caller` to the record (contains file name and line number like `foo.go:12`)
// Uses the `log15.CallerFileHandler` to perform this task
func CallerFileHandler(h LogHandler) LogHandler {
	return log15.CallerFileHandler(h)
}

// Adds in a context called `caller` to the record (contains file name and line number like `foo.go:12`)
// Uses the `log15.CallerFuncHandler` to perform this task
func CallerFuncHandler(h LogHandler) LogHandler {
	return log15.CallerFuncHandler(h)
}

// Filters out records which match the key value pair
// Uses the `log15.MatchFilterHandler` to perform this task
func MatchHandler(key string, value interface{}, h LogHandler) LogHandler {
	return log15.MatchFilterHandler(key, value, h)
}

// If match then A handler is called otherwise B handler is called
func MatchAbHandler(key string, value interface{}, a, b LogHandler) LogHandler {
	return log15.FuncHandler(func(r *log15.Record) error {
		for i := 0; i < len(r.Ctx); i += 2 {
			if r.Ctx[i] == key {
				if r.Ctx[i+1] == value {
					if a != nil {
						return a.Log(r)
					}
					return nil
				}
			}
		}
		if b != nil {
			return b.Log(r)
		}
		return nil
	})
}

// The nil handler is used if logging for a specific request needs to be turned off
func NilHandler() LogHandler {
	return log15.FuncHandler(func(r *log15.Record) error {
		return nil
	})
}

// Match all values in map to log
func MatchMapHandler(matchMap map[string]interface{}, a LogHandler) LogHandler {
	return matchMapHandler(matchMap, false, a)
}

// Match !(Match all values in map to log) The inverse of MatchMapHandler
func NotMatchMapHandler(matchMap map[string]interface{}, a LogHandler) LogHandler {
	return matchMapHandler(matchMap, true, a)
}

// Rather then chaining multiple filter handlers, process all here
func matchMapHandler(matchMap map[string]interface{}, inverse bool, a LogHandler) LogHandler {
	return log15.FuncHandler(func(r *log15.Record) error {
		checkMap := map[string]bool{}
		// Copy the map to a bool
		for i := 0; i < len(r.Ctx); i += 2 {
			if value, found := matchMap[r.Ctx[i].(string)]; found && value == r.Ctx[i+1] {
				checkMap[r.Ctx[i].(string)] = true
			}
		}
		if len(checkMap) == len(matchMap) {
			if !inverse {
				return a.Log(r)
			}
		} else if inverse {
			return a.Log(r)
		}
		return nil
	})
}

// Filters out records which do not match the key value pair
// Uses the `log15.FilterHandler` to perform this task
func NotMatchHandler(key string, value interface{}, h LogHandler) LogHandler {
	return log15.FilterHandler(func(r *log15.Record) (pass bool) {
		switch key {
		case r.KeyNames.Lvl:
			return r.Lvl != value
		case r.KeyNames.Time:
			return r.Time != value
		case r.KeyNames.Msg:
			return r.Msg != value
		}

		for i := 0; i < len(r.Ctx); i += 2 {
			if r.Ctx[i] == key {
				return r.Ctx[i+1] == value
			}
		}
		return true
	}, h)
}

func MultiHandler(hs ...LogHandler) LogHandler {
	// Convert the log handlers to log15.Handlers
	handlers := []log15.Handler{}
	for _, h := range hs {
		if h != nil {
			handlers = append(handlers, h)
		}
	}

	return log15.MultiHandler(handlers...)
}

// Outputs the records to the passed in stream
// Uses the `log15.StreamHandler` to perform this task
func StreamHandler(wr io.Writer, fmtr LogFormat) LogHandler {
	return log15.StreamHandler(wr, fmtr)
}

// Filter handler, this is the only
// Uses the `log15.FilterHandler` to perform this task
func FilterHandler(fn func(r *log15.Record) bool, h LogHandler) LogHandler {
	return log15.FilterHandler(fn, h)
}

type ListLogHandler struct {
	handlers []LogHandler
}

func NewListLogHandler(h1, h2 LogHandler) *ListLogHandler {
	ll := &ListLogHandler{handlers: []LogHandler{h1, h2}}
	return ll
}
func (ll *ListLogHandler) Log(r *log15.Record) (err error) {
	for _, handler := range ll.handlers {
		if err == nil {
			err = handler.Log(r)
		} else {
			handler.Log(r)
		}
	}
	return
}
func (ll *ListLogHandler) Add(h LogHandler) {
	if h != nil {
		ll.handlers = append(ll.handlers, h)
	}
}
func (ll *ListLogHandler) Del(h LogHandler) {
	if h != nil {
		for i, handler := range ll.handlers {
			if handler == h {
				ll.handlers = append(ll.handlers[:i], ll.handlers[i+1:]...)
			}
		}
	}
}

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
func (h *CompositeMultiHandler) Log(r *log15.Record) (err error) {

	var handler LogHandler
	switch r.Lvl {
	case log15.LvlInfo:
		handler = h.InfoHandler
	case log15.LvlDebug:
		handler = h.DebugHandler
	case log15.LvlWarn:
		handler = h.WarnHandler
	case log15.LvlError:
		handler = h.ErrorHandler
	case log15.LvlCrit:
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
	handler := CallerFileHandler(StreamHandler(writer, log15.JsonFormatEx(
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
	handler := CallerFileHandler(StreamHandler(
		writer,
		TerminalFormatHandler(
			options.GetBoolDefault("noColor", false),
			options.GetBoolDefault("smallDate", true))))
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
