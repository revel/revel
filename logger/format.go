package logger

import (
	"bytes"
	"fmt"
	"github.com/revel/log15"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	timeFormat          = "2006-01-02T15:04:05-0700"
	termTimeFormat      = "2006/01/02 15:04:05"
	termSmallTimeFormat = "15:04:05"
	floatFormat         = 'f'
	errorKey            = "REVEL_ERROR"
)

var (
	// Name the log level
	toRevel = map[log15.Lvl]string{log15.LvlDebug: "DEBUG",
		log15.LvlInfo: "INFO", log15.LvlWarn: "WARN", log15.LvlError: "ERROR", log15.LvlCrit: "CRIT"}
)

// Outputs to the terminal in a format like below
// INFO  09:11:32 server-engine.go:169: Request Stats
func TerminalFormatHandler(noColor bool, smallDate bool) LogFormat {
	dateFormat := termTimeFormat
	if smallDate {
		dateFormat = termSmallTimeFormat
	}
	return log15.FormatFunc(func(r *log15.Record) []byte {
		// Bash coloring http://misc.flogisoft.com/bash/tip_colors_and_formatting
		var color = 0
		switch r.Lvl {
		case log15.LvlCrit:
			// Magenta
			color = 35
		case log15.LvlError:
			// Red
			color = 31
		case log15.LvlWarn:
			// Yellow
			color = 33
		case log15.LvlInfo:
			// Green
			color = 32
		case log15.LvlDebug:
			// Cyan
			color = 36
		}

		b := &bytes.Buffer{}
		lvl := strings.ToUpper(r.Lvl.String())
		caller := findInContext("caller", r.Ctx)
		module := findInContext("module", r.Ctx)
		if noColor == false && color > 0 {
			if len(module) > 0 {
				fmt.Fprintf(b, "\x1b[%dm%-5s\x1b[0m %s %6s %13s: %-40s ", color, toRevel[r.Lvl], r.Time.Format(dateFormat), module, caller, r.Msg)
			} else {
				fmt.Fprintf(b, "\x1b[%dm%-5s\x1b[0m %s %13s: %-40s ", color, toRevel[r.Lvl], r.Time.Format(dateFormat), caller, r.Msg)
			}
		} else {
			fmt.Fprintf(b, "%-5s %s %6s %13s: %-40s", toRevel[r.Lvl], r.Time.Format(dateFormat), module, caller, r.Msg)
			fmt.Fprintf(b, "[%s] [%s] %s ", lvl, r.Time.Format(dateFormat), r.Msg)
		}

		for i := 0; i < len(r.Ctx); i += 2 {
			if i != 0 {
				b.WriteByte(' ')
			}

			k, ok := r.Ctx[i].(string)
			if k == "caller" || k == "fn" || k == "module" {
				continue
			}
			v := formatLogfmtValue(r.Ctx[i+1])
			if !ok {
				k, v = errorKey, formatLogfmtValue(k)
			}

			// TODO: we should probably check that all of your key bytes aren't invalid
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
		return strconv.FormatFloat(v, floatFormat, 7, 64)
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
