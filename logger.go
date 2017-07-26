package revel

import (
	"bytes"
	"fmt"
	"github.com/agtorre/gocolorize"
	"gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/inconshreveable/log15.v2/stack"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// This pacakge is for all log specific

// The Multilogger reduces the number of exposed defined logging variables,
// and allows the output to be easily refined
type MultiLogger interface {
	log15.Logger
}
type Handler interface {
	log15.Handler
}

//Logger
var (
	// This Log is the revel logger, historical for revel, The application should use the AppLog or the Controller.Log
	Log = log15.New().(MultiLogger)
	// This logger is the application logger, use this for your application log messages - ie jobs and startup,
	// Use Controller.Log for Controller logging
	AppLog        = log15.New().(MultiLogger)
	appLogHandler Handler

	toLevel = map[string]log15.Lvl{"trace": log15.LvlDebug,
		"info": log15.LvlInfo, "request": log15.LvlInfo, "warn": log15.LvlWarn, "error": log15.LvlError}

	toRevel = map[log15.Lvl]string{log15.LvlDebug: "TRACE",
		log15.LvlInfo: "INFO", log15.LvlWarn: "WARN", log15.LvlError: "ERROR"}

	// Loggers used in revel
	// DEPRECATED Use AppLog
	TRACE = getLogger("trace")
	// DEPRECATED Use AppLog
	INFO = getLogger("info")
	// DEPRECATED Use AppLog
	WARN = getLogger("warn")
	// DEPRECATED Use AppLog
	ERROR = getLogger("error")
)

func InitLogger() {
	// Initialize the logger and handler to dump to the terminal
	Log.SetHandler(log15.LvlFilterHandler(log15.LvlDebug, log15.StreamHandler(os.Stdout, terminalFormat(false))))

	if runtime.GOOS == "windows" {
		gocolorize.SetPlain(true)
	}
	OnAppStart(afterAppInitialized, -1)
}

// Set revel log
func SetLog(mainLogger MultiLogger) {
	Log = mainLogger
	TRACE = getLogger("trace")
	INFO = getLogger("info")
	WARN = getLogger("warn")
	ERROR = getLogger("error")
	afterAppInitialized()
}

// Set handlers for all the loggers, allows you to set custom handlers for all the loggers
// implement revel.Handler to set
func SetLogHandlers(trace, info, warn, error Handler) {
	Log.SetHandler(log15.MultiHandler(trace, info, warn, error))
}

// Set the application log
func SetAppLog(appLog MultiLogger) {
	AppLog = appLog
	AppLog.SetHandler(appLogHandler)
}

// Set the handler for the application log
func SetAppLogHandlers(app Handler) {
	appLogHandler = app
	AppLog.SetHandler(app)
}

// Initialize all handlers based on the Config (if available)
func afterAppInitialized() {
	h := map[string]log15.Handler{}
	noColor := true
	maxSize := 1024 * 10
	maxAge := 14
	if Config != nil {
		noColor = !Config.BoolDefault("log.colorize", true)
		maxSize = Config.IntDefault("log.maxsize", 1024*10)
		maxAge = Config.IntDefault("log.maxage", 14)
	}
	for _, name := range []string{"trace", "info", "warn", "error", "request"} {
		output := Config.StringDefault("log."+name+".output", "stderr")

		switch output {
		case "stdout":
			h[name] = LvlFilterHandler(toLevel[name], log15.StreamHandler(os.Stdout, terminalFormat(noColor)))
		case "stderr":
			h[name] = LvlFilterHandler(toLevel[name], log15.StreamHandler(os.Stderr, terminalFormat(noColor)))
		default:
			// Write to file specified
			if !filepath.IsAbs(output) {
				output = filepath.Join(BasePath, output)
			}

			logPath := filepath.Dir(output)
			if err := createDir(logPath); err != nil {
				log.Fatalln(err)
			}

			// If file name ends in json output in json format, use lumberjac.logger to rotate files
			if strings.HasSuffix(logPath, "json") {
				h[name] = LvlFilterHandler(toLevel[name], log15.StreamHandler(&lumberjack.Logger{
					Filename: logPath,
					MaxSize:  maxSize, // megabytes
					MaxAge:   maxAge,  //days
				}, log15.JsonFormatEx(false, true)))
			} else {
				h[name] = LvlFilterHandler(toLevel[name], log15.StreamHandler(&lumberjack.Logger{
					Filename: logPath,
					MaxSize:  maxSize, // megabytes
					MaxAge:   maxAge,  //days
				}, terminalFormat(true)))

			}
		}

	}

	// Set all the log handlers
	SetLogHandlers(h["trace"], h["info"], h["warn"], h["error"])
	SetAppLogHandlers(h["request"])
}

func getLogger(name string) (l *log.Logger) {
	switch name {
	case "trace":
		l = log.New(loggerRewrite{Logger: Log, Level: log15.LvlInfo}, "", 0)
	case "info":
		l = log.New(loggerRewrite{Logger: Log, Level: log15.LvlDebug}, "", 0)
	case "warn":
		l = log.New(loggerRewrite{Logger: Log, Level: log15.LvlWarn}, "", 0)
	case "error":
		l = log.New(loggerRewrite{Logger: Log, Level: log15.LvlError}, "", 0)
	case "request":
		l = log.New(loggerRewrite{Logger: Log, Level: log15.LvlInfo}, "", 0)
	}

	return l

}

// This structure and method will handle the old output format and log it to the new format
type loggerRewrite struct {
	Logger MultiLogger
	Level  log15.Lvl
}

func (s loggerRewrite) Write(p []byte) (n int, err error) {
	n = len(p)
	var CallPC [3]uintptr
	runtime.Callers(3, CallPC[:])

	call := stack.Call(CallPC[2])
	caller := fmt.Sprint(call)
	fn := fmt.Sprintf("%+n", call)
	if i := strings.Index(fn, "/"); i > 0 {
		fn = fn[i:]
	}
	if i := strings.Index(fn, "."); i > 0 {
		fn = fn[i:]
	}
	if len(p) > 0 && p[n-1] == '\n' {
		p = p[:n-1]
	}

	switch s.Level {
	case log15.LvlInfo:
		s.Logger.Info(string(p), "caller", caller, "fn", fn)
	case log15.LvlDebug:
		s.Logger.Debug(string(p), "caller", caller, "fn", fn)
	case log15.LvlWarn:
		s.Logger.Warn(string(p), "caller", caller, "fn", fn)
	case log15.LvlError:
		s.Logger.Error(string(p), "caller", caller, "fn", fn)
	case log15.LvlCrit:
		s.Logger.Crit(string(p), "caller", caller, "fn", fn)
	}

	return
}

const (
	timeFormat     = "2006-01-02T15:04:05-0700"
	termTimeFormat = "2006/01/02 15:04:05"
	floatFormat    = 'f'
	termMsgJust    = 40
	errorKey       = "LOG15_ERROR"
)

// INFO  2017/07/25 14:53:40 db.go:768: [gorp] begin; [] (138.433µs)
func terminalFormat(noColor bool) log15.Format {
	return log15.FormatFunc(func(r *log15.Record) []byte {
		var color = 0
		switch r.Lvl {
		case log15.LvlCrit:
			color = 35
		case log15.LvlError:
			color = 31
		case log15.LvlWarn:
			color = 33
		case log15.LvlInfo:
			color = 32
		case log15.LvlDebug:
			color = 36
		}

		b := &bytes.Buffer{}
		lvl := strings.ToUpper(r.Lvl.String())
		if noColor == false && color > 0 {
			// fn := findInContext("fn",r.Ctx)
			caller := findInContext("caller", r.Ctx)

			fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m %s %s: %s ", color, toRevel[r.Lvl], r.Time.Format(termTimeFormat), caller, r.Msg)

		} else {
			fmt.Fprintf(b, "[%s] [%s] %s ", lvl, r.Time.Format(termTimeFormat), r.Msg)
		}

		// try to justify the log output for short messages
		if len(r.Ctx) > 0 && len(r.Msg) < termMsgJust {
			b.Write(bytes.Repeat([]byte{' '}, termMsgJust-len(r.Msg)))
		}

		for i := 0; i < len(r.Ctx); i += 2 {
			if i != 0 {
				b.WriteByte(' ')
			}

			k, ok := r.Ctx[i].(string)
			if k == "caller" || k == "fn" {
				continue
			}
			v := formatLogfmtValue(r.Ctx[i+1])
			if !ok {
				k, v = errorKey, formatLogfmtValue(k)
			}

			// XXX: we should probably check that all of your key bytes aren't invalid
			if color > 0 {
				fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m=%s", color, k, v)
			} else {
				b.WriteString(k)
				b.WriteByte('=')
				b.WriteString(v)
			}
		}

		b.WriteByte('\n')

		return b.Bytes()
	})
}
func findInContext(key string, ctx []interface{}) string {
	for i := 0; i < len(ctx); i += 2 {
		k := ctx[i].(string)
		if key == k {
			return formatLogfmtValue(ctx[i+1])
		}
	}
	return ""
}

// formatValue formats a value for serialization
func formatLogfmtValue(value interface{}) string {
	if value == nil {
		return "nil"
	}

	if t, ok := value.(time.Time); ok {
		// Performance optimization: No need for escaping since the provided
		// timeFormat doesn't have any escape characters, and escaping is
		// expensive.
		return t.Format(termTimeFormat)
	}
	value = formatShared(value)
	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case float32:
		return strconv.FormatFloat(float64(v), floatFormat, 3, 64)
	case float64:
		return strconv.FormatFloat(v, floatFormat, 3, 64)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", value)
	case string:
		return escapeString(v)
	default:
		return escapeString(fmt.Sprintf("%+v", value))
	}
}
func formatShared(value interface{}) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
				result = "nil"
			} else {
				panic(err)
			}
		}
	}()

	switch v := value.(type) {
	case time.Time:
		return v.Format(timeFormat)

	case error:
		return v.Error()

	case fmt.Stringer:
		return v.String()

	default:
		return v
	}
}

var stringBufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func escapeString(s string) string {
	needsQuotes := false
	needsEscape := false
	for _, r := range s {
		if r <= ' ' || r == '=' || r == '"' {
			needsQuotes = true
		}
		if r == '\\' || r == '"' || r == '\n' || r == '\r' || r == '\t' {
			needsEscape = true
		}
	}
	if needsEscape == false && needsQuotes == false {
		return s
	}
	e := stringBufPool.Get().(*bytes.Buffer)
	e.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\', '"':
			e.WriteByte('\\')
			e.WriteByte(byte(r))
		case '\n':
			e.WriteString("\\n")
		case '\r':
			e.WriteString("\\r")
		case '\t':
			e.WriteString("\\t")
		default:
			e.WriteRune(r)
		}
	}
	e.WriteByte('"')
	var ret string
	if needsQuotes {
		ret = e.String()
	} else {
		ret = string(e.Bytes()[1 : e.Len()-1])
	}
	e.Reset()
	stringBufPool.Put(e)
	return ret
}

func LvlFilterHandler(lvl log15.Lvl, h log15.Handler) log15.Handler {
	return log15.FilterHandler(func(r *log15.Record) (pass bool) {
		return r.Lvl == lvl
	}, h)
}
