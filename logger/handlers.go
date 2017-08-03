package logger

import (
	"gopkg.in/inconshreveable/log15.v2"
	"io"
)

// Filters out records which do not match the level
// Uses the `log15.FilterHandler` to perform this task
func LevelHandler(lvl log15.Lvl, h LogHandler) LogHandler {
	return log15.FilterHandler(func(r *log15.Record) (pass bool) {
		return r.Lvl == lvl
	}, h)
}

// Filters out records which match the level
// Uses the `log15.FilterHandler` to perform this task
func NotLevelHandler(lvl log15.Lvl, h LogHandler) LogHandler {
	return log15.FilterHandler(func(r *log15.Record) (pass bool) {
		return r.Lvl != lvl
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
	return log15.MatchFilterHandler(key,value,h)
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
	for _ ,h:= range hs {
		if h!=nil {
			handlers = append(handlers, h)
		}
	}

	return log15.MultiHandler(handlers...)
}

// Outputs the records to the passed in stream
// Uses the `log15.StreamHandler` to perform this task
func StreamHandler(wr io.Writer, fmtr LogFormat) LogHandler {
	return log15.StreamHandler(wr,fmtr)
}
// Filter handler, this is the only
// Uses the `log15.FilterHandler` to perform this task
func FilterHandler(fn func(r *log15.Record) bool, h LogHandler) LogHandler {
	return log15.FilterHandler(fn,h)
}