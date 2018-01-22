package logger

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type Builder struct {
	Debug    []zapcore.Core
	Info     []zapcore.Core
	Warn     []zapcore.Core
	Error    []zapcore.Core
	Critical []zapcore.Core
}

func NewBuilder() *Builder {
	return &Builder{}
}
func (h *Builder) SetHandler(handler zapcore.Core, replace bool, level LogLevel) {
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
		*source = []zapcore.Core{handler}
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
		h.SetHandler(core, options.ReplaceExistingHandler, lvl)
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

func (h *Builder) Build() MultiLogger {
	logger := zap.New(cores{
		debug: zapcore.NewTee(h.Debug...),
		info:  zapcore.NewTee(h.Info...),
		warn:  zapcore.NewTee(h.Warn...),
		err:   zapcore.NewTee(h.Error...),
	}).WithOptions(zap.AddCallerSkip(1))

	return &RevelLogger{l: logger, s: logger.Sugar()}
}

const (
	matchSkip = iota
	matchTrue
	matchFalse
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
		return matchSkip
	default:
		return matchSkip
	}
	if matchResult {
		return matchTrue
	}
	return matchFalse
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
			if mr == matchTrue {
				matchedKeys = append(matchedKeys, f.Key)
			} else if mr == matchFalse {
				if p.inverse {
					return p.impl.With(fields)
				}
				return zapcore.NewNopCore()
			}
		}
	}

	if len(matchedKeys) == len(p.matchValues) {
		if p.inverse {
			return zapcore.NewNopCore()
		}
		return p.impl.With(fields)
	}

	matchValues := p.matchValues
	if len(matchedKeys) > 0 {
		matchValues = make(map[string]string, len(p.matchValues))
		for k, v := range p.matchValues {
			found := false
			for _, key := range matchedKeys {
				if key == k {
					found = true
					break
				}
			}

			if !found {
				matchValues[k] = v
			}
		}
	}

	return proxyCore{impl: p.impl.With(fields),
		matchValues: matchValues,
		inverse:     p.inverse}
}
func (p proxyCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if p.Enabled(entry.Level) {
		return ce.AddCore(entry, p)
	}
	return ce
}
func (p proxyCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	var matchedKeys []string
	for _, f := range fields {
		excepted, ok := p.matchValues[f.Key]
		if ok {
			mr := matchField(f, excepted)
			if mr == matchTrue {
				matchedKeys = append(matchedKeys, f.Key)
			} else if mr == matchFalse {
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

type abCore struct {
	a, b       zapcore.Core
	key, value string
}

func (ab abCore) Enabled(l zapcore.Level) bool {
	return ab.Enabled(l)
}

func (ab abCore) With(fields []zapcore.Field) zapcore.Core {
	for _, f := range fields {
		if f.Key == ab.key {
			mr := matchField(f, ab.value)
			if mr == matchTrue {
				return ab.a.With(fields)
			} else if mr == matchFalse {
				return ab.b.With(fields)
			}
		}
	}

	return abCore{a: ab.a.With(fields),
		b:     ab.b.With(fields),
		key:   ab.key,
		value: ab.value}
}

func (ab abCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if ab.a.Enabled(entry.Level) || ab.b.Enabled(entry.Level) {
		return ce.AddCore(entry, ab)
	}
	return ce
}

func (ab abCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	for _, f := range fields {
		if f.Key == ab.key {
			mr := matchField(f, ab.value)
			if mr == matchTrue {
				return ab.a.Write(entry, fields)
			} else if mr == matchFalse {
				return ab.b.Write(entry, fields)
			}
		}
	}
	return ab.b.Write(entry, fields)
}
func (ab abCore) Sync() error {
	err1 := ab.a.Sync()
	err2 := ab.b.Sync()
	return multierr.Append(err1, err2)
}

type cores struct {
	debug zapcore.Core
	info  zapcore.Core
	warn  zapcore.Core
	err   zapcore.Core
}

func (c cores) Enabled(l zapcore.Level) bool {
	switch l {
	case zapcore.DebugLevel:
		return c.debug.Enabled(l)
	case zapcore.InfoLevel:
		return c.info.Enabled(l)
	case zapcore.WarnLevel:
		return c.warn.Enabled(l)
	case zapcore.ErrorLevel:
		return c.err.Enabled(l)
	default:
		return c.err.Enabled(l)
	}
}

func (c cores) With(fields []zapcore.Field) zapcore.Core {
	return cores{
		debug: c.debug.With(fields),
		info:  c.info.With(fields),
		warn:  c.warn.With(fields),
		err:   c.err.With(fields)}
}

func (c cores) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	switch entry.Level {
	case zapcore.DebugLevel:
		return c.debug.Check(entry, ce)
	case zapcore.InfoLevel:
		return c.info.Check(entry, ce)
	case zapcore.WarnLevel:
		return c.warn.Check(entry, ce)
	case zapcore.ErrorLevel:
		return c.err.Check(entry, ce)
	default:
		return c.err.Check(entry, ce)
	}
}

func (c cores) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// this method will not invoke.

	// switch entry.Level {
	// case zapcore.DebugLevel:
	// 	return c.debug.Write(entry, fields)
	// case zapcore.InfoLevel:
	// 	return c.info.Write(entry, fields)
	// case zapcore.WarnLevel:
	// 	return c.warn.Write(entry, fields)
	// case zapcore.ErrorLevel:
	// 	return c.err.Write(entry, fields)
	// default:
	// 	return c.err.Write(entry, fields)
	// }
	panic(errors.New("cores.Write() will not invoke"))
}
func (c cores) Sync() error {
	// this method will not invoke.
	panic(errors.New("cores.Sync() will not invoke"))

	// err := c.debug.Sync()
	// err = multierr.Append(err, c.info.Sync())
	// err = multierr.Append(err, c.warn.Sync())
	// return multierr.Append(err, c.err.Sync())
}
