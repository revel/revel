package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
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
	levelString = map[LogLevel]string{LvlDebug: "DEBUG",
		LvlInfo: "INFO", LvlWarn: "WARN", LvlError: "ERROR", LvlCrit: "CRIT"}
)

// Outputs to the terminal in a format like below
// INFO  09:11:32 server-engine.go:169: Request Stats
func TerminalFormatHandler(noColor bool, smallDate bool) LogFormat {
	dateFormat := termTimeFormat
	if smallDate {
		dateFormat = termSmallTimeFormat
	}
	return FormatFunc(func(r *Record) []byte {
		// Bash coloring http://misc.flogisoft.com/bash/tip_colors_and_formatting
		var color = 0
		switch r.Level {
		case LvlCrit:
			// Magenta
			color = 35
		case LvlError:
			// Red
			color = 31
		case LvlWarn:
			// Yellow
			color = 33
		case LvlInfo:
			// Green
			color = 32
		case LvlDebug:
			// Cyan
			color = 36
		}

		b := &bytes.Buffer{}
		caller, _ := r.Context["caller"].(string)
		module, _ := r.Context["module"].(string)
		if noColor == false && color > 0 {
			if len(module) > 0 {
				fmt.Fprintf(b, "\x1b[%dm%-5s\x1b[0m %s %6s %13s: %-40s ", color, levelString[r.Level], r.Time.Format(dateFormat), module, caller, r.Message)
			} else {
				fmt.Fprintf(b, "\x1b[%dm%-5s\x1b[0m %s %13s: %-40s ", color, levelString[r.Level], r.Time.Format(dateFormat), caller, r.Message)
			}
		} else {
			fmt.Fprintf(b, "%-5s %s %6s %13s: %-40s", levelString[r.Level], r.Time.Format(dateFormat), module, caller, r.Message)
		}

		i := 0
		for k, v := range r.Context {
			if i != 0 {
				b.WriteByte(' ')
			}
			i++
			if k == "module" || k == "caller" {
				continue
			}

			v := formatLogfmtValue(v)

			// TODO: we should probably check that all of your key bytes aren't invalid
			if noColor == false && color > 0 {
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

// Format the value in json format
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

// A reusuable buffer for outputting data
var stringBufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// Escape the string when needed
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

// JsonFormatEx formats log records as JSON objects. If pretty is true,
// records will be pretty-printed. If lineSeparated is true, records
// will be logged with a new line between each record.
func JsonFormatEx(pretty, lineSeparated bool) LogFormat {
	jsonMarshal := json.Marshal
	if pretty {
		jsonMarshal = func(v interface{}) ([]byte, error) {
			return json.MarshalIndent(v, "", "    ")
		}
	}

	return FormatFunc(func(r *Record) []byte {
		props := make(map[string]interface{})

		props["t"] = r.Time
		props["lvl"] = levelString[r.Level]
		props["msg"] = r.Message
		for k, v := range r.Context {
			props[k] = formatJsonValue(v)
		}

		b, err := jsonMarshal(props)
		if err != nil {
			b, _ = jsonMarshal(map[string]string{
				errorKey: err.Error(),
			})
			return b
		}

		if lineSeparated {
			b = append(b, '\n')
		}

		return b
	})
}

func formatJsonValue(value interface{}) interface{} {
	value = formatShared(value)
	switch value.(type) {
	case int, int8, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64, string:
		return value
	default:
		return fmt.Sprintf("%+v", value)
	}
}
