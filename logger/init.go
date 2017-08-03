package logger

import (

	"gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"os"
	"path/filepath"
	"strings"
	"github.com/revel/config"
)
var (

	toLevel = map[string]log15.Lvl{"trace": log15.LvlDebug,
		"info": log15.LvlInfo, "request": log15.LvlInfo, "warn": log15.LvlWarn,
		"error": log15.LvlError,"crit": log15.LvlCrit}

	toRevel = map[log15.Lvl]string{log15.LvlDebug: "TRACE",
		log15.LvlInfo: "INFO", log15.LvlWarn: "WARN", log15.LvlError: "ERROR", log15.LvlCrit: "CRIT"}

)

func GetLogger(name string, logger MultiLogger) (l *log.Logger) {
	switch name {
	case "trace":
		l = log.New(loggerRewrite{Logger: logger, Level: log15.LvlDebug}, "", 0)
	case "info":
		l = log.New(loggerRewrite{Logger: logger, Level: log15.LvlInfo}, "", 0)
	case "warn":
		l = log.New(loggerRewrite{Logger: logger, Level: log15.LvlWarn}, "", 0)
	case "error":
		l = log.New(loggerRewrite{Logger: logger, Level: log15.LvlError}, "", 0)
	case "request":
		l = log.New(loggerRewrite{Logger: logger, Level: log15.LvlInfo}, "", 0)
	}

	return l

}



// Get all handlers based on the Config (if available)
func GetHandlers(basePath string, config *config.Context) (trace, info, warn, error, app LogHandler) {
	h := map[string]log15.Handler{}
	appHandlers := []LogHandler{}
	for _, name := range []string{"trace", "info", "warn", "error","crit"} {
		output := "stderr"
		if config !=nil {
			config.StringDefault("log."+name+".output", "stderr")
		}
		outputApp := output
		if config!=nil {
			config.StringDefault("log.app."+name+".output", output)
		}
		if handler:=handlerFor(name,output, basePath, config);handler!=nil {
			h[name] = handler
		}
		if handler:=handlerFor(name, outputApp, basePath, config);handler!=nil {
			appHandlers = append(appHandlers, handler)
		}
	}

	// Backwards compatibility for request logging
	outputRequest := "stdout"
	if config!=nil {
		config.StringDefault("log.request.output", "stdout")
	}
	if outputRequest!="" {
		if handler:=handlerFor("info",outputRequest, basePath, config);handler!=nil {
			appHandlers = append(appHandlers, MatchHandler("type","request",handler))
		}

	}

	// Set all the log handlers
	return h["trace"], h["info"], h["warn"], h["error"], MultiHandler(appHandlers...)
}

// Returns a handler for the level using the output string
// Accept formats for output string are
// `stdout` `stderr` `full/file/path/to/location/app.log` `full/file/path/to/location/app.json`
func handlerFor(level, output, basePath string, config *config.Context) (h LogHandler){
	noColor := false
	maxSize := 1024 * 10
	maxAge := 14
	if config != nil {
		noColor = !config.BoolDefault("log.colorize", true)
		maxSize = config.IntDefault("log.maxsize", 1024*10)
		maxAge = config.IntDefault("log.maxage", 14)
	}

	switch strings.TrimSpace(output) {
	case "":
		fallthrough
	case "off":
		// No handler, discard data
	case "stdout":
		h = LevelHandler(toLevel[level], log15.StreamHandler(os.Stdout, TerminalFormatHandler(noColor, true)))
	case "stderr":
		h = LevelHandler(toLevel[level], log15.StreamHandler(os.Stderr, TerminalFormatHandler(noColor, true)))
	default:
		log.Println("oldLog file being created ","name", output,"level",level)
		// Write to file specified
		if !filepath.IsAbs(output) {
			output = filepath.Join(basePath, output)
		}

		if err := os.MkdirAll(filepath.Dir(output),0755); err != nil {
			log.Panic(err)
		}

		// If file name ends in json output in json format, use lumberjac.logger to rotate files
		if strings.HasSuffix(output, "json") {
			h = LevelHandler(toLevel[level], log15.StreamHandler(&lumberjack.Logger{
				Filename: output,
				MaxSize:  maxSize, // megabytes
				MaxAge:   maxAge,  //days
			}, log15.JsonFormatEx(false, true)))
		} else {
			h = LevelHandler(toLevel[level], log15.StreamHandler(&lumberjack.Logger{
				Filename: output,
				MaxSize:  maxSize, // megabytes
				MaxAge:   maxAge,  //days
			}, TerminalFormatHandler(true, false)))
		}
	}
	return
}


// This structure and method will handle the old output format and log it to the new format
type loggerRewrite struct {
	Logger MultiLogger
	Level  log15.Lvl
	hideDeprecated bool
}

var log_deprecated=[]byte("* LOG DEPRECATED * ")
func (lr loggerRewrite) Write(p []byte) (n int, err error) {
	if !lr.hideDeprecated {
		p = append(log_deprecated, p...)
	}
	n=len(p)
	if len(p) > 0 && p[n-1] == '\n' {
		p = p[:n-1]
		n --
	}

	switch lr.Level {
	case log15.LvlInfo:
		lr.Logger.Info(string(p))
	case log15.LvlDebug:
		lr.Logger.Debug(string(p))
	case log15.LvlWarn:
		lr.Logger.Warn(string(p))
	case log15.LvlError:
		lr.Logger.Error(string(p))
	case log15.LvlCrit:
		lr.Logger.Crit(string(p))
	}

	return
}