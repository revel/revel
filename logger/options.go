package logger

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/revel/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
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
			zapcore.Level(lvl),
		)

		if options.HandlerWrap != nil {
			core = options.HandlerWrap(core, options)
		}
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
			// case LvlCrit:
			// 	h.Critical = nil
		}
	}
}

const (
	MatchSkip = iota
	MatchTrue
	MatchFalse
)

func matchBinary(bs []byte, value string) bool {
	return string(bs) == value
}

func matchBoolean(b bool, value string) bool {
	if b {
		return "true" == value
	}
	return "true" != value
}

func matchDuration(duration time.Duration, value string) bool {
	d, err := time.ParseDuration(value)
	if err != nil {
		return false
	}
	return duration == d
}

func matchTime(t time.Time, value string) bool {
	tv, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return false
	}
	return t == tv
}

func matchInt64(i int64, value string) bool {
	return strconv.FormatInt(i, 10) == value
}

func matchUint64(i uint64, value string) bool {
	return strconv.FormatUint(i, 10) == value
}

func matchString(s string, value string) bool {
	return s == value
}

func matchField(f zapcore.Field, value string) int {
	var matchResult bool
	switch f.Type {
	// case ArrayMarshalerType:
	// 	err = enc.AddArray(f.Key, f.Interface.(ArrayMarshaler))
	// case ObjectMarshalerType:
	// 	err = enc.AddObject(f.Key, f.Interface.(ObjectMarshaler))
	case zapcore.BinaryType:
		matchResult = matchBinary(f.Interface.([]byte), value)
	case zapcore.BoolType:
		matchResult = matchBoolean(f.Integer == 1, value)
	case zapcore.ByteStringType:
		matchResult = matchBinary(f.Interface.([]byte), value)
	// case Complex128Type:
	// 	enc.AddComplex128(f.Key, f.Interface.(complex128))
	// case Complex64Type:
	// 	enc.AddComplex64(f.Key, f.Interface.(complex64))
	case zapcore.DurationType:
		matchResult = matchDuration(time.Duration(f.Integer), value)
	// case Float64Type:
	// 	enc.AddFloat64(f.Key, math.Float64frombits(uint64(f.Integer)))
	// case Float32Type:
	// 	enc.AddFloat32(f.Key, math.Float32frombits(uint32(f.Integer)))
	case zapcore.Int64Type:
		matchResult = matchInt64(f.Integer, value)
	case zapcore.Int32Type:
		matchResult = matchInt64(f.Integer, value)
	case zapcore.Int16Type:
		matchResult = matchInt64(f.Integer, value)
	case zapcore.Int8Type:
		matchResult = matchInt64(f.Integer, value)
	case zapcore.StringType:
		matchResult = matchString(f.String, value)
	case zapcore.TimeType:
		if f.Interface != nil {
			matchResult = matchTime(time.Unix(0, f.Integer).In(f.Interface.(*time.Location)), value)
		} else {
			// Fall back to UTC if location is nil.
			matchResult = matchTime(time.Unix(0, f.Integer), value)
		}
	case zapcore.Uint64Type:
		matchResult = matchUint64(uint64(f.Integer), value)
	case zapcore.Uint32Type:
		matchResult = matchUint64(uint64(f.Integer), value)
	case zapcore.Uint16Type:
		matchResult = matchUint64(uint64(f.Integer), value)
	case zapcore.Uint8Type:
		matchResult = matchUint64(uint64(f.Integer), value)
	// case UintptrType:
	// 	return matchUint64(uint64(f.Integer), value)
	// case ReflectType:
	// 	err = enc.AddReflected(f.Key, f.Interface)
	// case NamespaceType:
	// 	enc.OpenNamespace(f.Key)
	case zapcore.StringerType:
		matchResult = matchString(f.Interface.(fmt.Stringer).String(), value)
	// case ErrorType:
	// 	encodeError(f.Key, f.Interface.(error), enc)
	case zapcore.SkipType:
		return MatchSkip
	default:
		return MatchSkip
	}
	if matchResult {
		return MatchTrue
	}
	return MatchFalse
}

type proxyCore struct {
	impl        zapcore.Core
	matchValues map[string]string
	inverse     bool
}

func (p proxyCore) Enabled(l zapcore.Level) bool {
	return p.Enabled(l)
}
func (p proxyCore) With(fields []zapcore.Field) zapcore.Core {
	var matchedKeys []string
	for _, f := range fields {
		excepted, ok := p.matchValues[f.Key]
		if ok {
			mr := matchField(f, excepted)
			if mr == MatchTrue {
				matchedKeys = append(matchedKeys, f.Key)
			} else if mr == MatchFalse {
				enabled := false
				if p.inverse {
					enabled = true
				}
				return enableCore{impl: p.impl.With(fields),
					enabled: enabled}
			}
		}
	}

	if len(matchedKeys) == len(p.matchValues) {
		enabled := true
		if p.inverse {
			enabled = false
		}
		return enableCore{impl: p.impl.With(fields),
			enabled: enabled}
	}

	if len(matchedKeys) > 0 {
		matchValues := make(map[string]string, len(p.matchValues))
		for k, v := range p.matchValues {
			matchValues[k] = v
		}
		for _, key := range matchedKeys {
			delete(matchValues, key)
		}
		return proxyCore{impl: p.impl.With(fields),
			matchValues: matchValues,
			inverse:     p.inverse}
	}

	return proxyCore{impl: p.impl.With(fields),
		matchValues: p.matchValues,
		inverse:     p.inverse}
}
func (p proxyCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return p.impl.Check(entry, ce)
}
func (p proxyCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	var matchedKeys []string
	for _, f := range fields {
		excepted, ok := p.matchValues[f.Key]
		if ok {
			mr := matchField(f, excepted)
			if mr == MatchTrue {
				matchedKeys = append(matchedKeys, f.Key)
			} else if mr == MatchFalse {
				goto notmatch
			}
		}
	}
	if len(matchedKeys) == len(p.matchValues) {
		if p.inverse {
			return nil
		}
		return p.impl.Write(entry, fields)
	}

notmatch:
	if p.inverse {
		return p.impl.Write(entry, fields)
	}
	return nil
}
func (p proxyCore) Sync() error {
	return p.impl.Sync()
}

type enableCore struct {
	impl    zapcore.Core
	enabled bool
}

func (p enableCore) Enabled(l zapcore.Level) bool {
	return p.Enabled(l) && p.enabled
}
func (p enableCore) With(fields []zapcore.Field) zapcore.Core {
	return enableCore{impl: p.impl.With(fields), enabled: p.enabled}
}
func (p enableCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return p.impl.Check(entry, ce)
}
func (p enableCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return p.impl.Write(entry, fields)
}
func (p enableCore) Sync() error {
	return p.impl.Sync()
}
