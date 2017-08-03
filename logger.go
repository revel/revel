package revel

import (
	"github.com/revel/revel/logger"
	"gopkg.in/inconshreveable/log15.v2"
	"log"
	"os"
)

//Logger
var (
	// This logger is the application logger, use this for your application log messages - ie jobs and startup,
	// Use Controller.oldLog for Controller logging
	// The requests are logged to this logger with the context of `type:request`
	AppLog = log15.New().(logger.MultiLogger)
	// This is the logger revel writes to, added log messages will have a context of module:revel in them
	// It is based off of `AppLog`
	RevelLog = AppLog.New("module", "Revel")

	// This is the handler for the AppLog, it is stored so that if the AppLog is changed it can be assigned to the
	// new AppLog
	appLogHandler logger.LogHandler

	// This oldLog is the revel logger, historical for revel, The application should use the AppLog or the Controller.oldLog
	// DEPRECATED
	oldLog = log15.New().(logger.MultiLogger)
	// DEPRECATED Use AppLog
	TRACE = log.New(os.Stdout, "TRACE ", log.Ldate|log.Ltime|log.Lshortfile)
	// DEPRECATED Use AppLog
	INFO = log.New(os.Stdout, "INFO ", log.Ldate|log.Ltime|log.Lshortfile)
	// DEPRECATED Use AppLog
	WARN = log.New(os.Stdout, "WARN ", log.Ldate|log.Ltime|log.Lshortfile)
	// DEPRECATED Use AppLog
	ERROR = log.New(os.Stderr, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile)
)

func init() {
	oldLog.SetHandler(logger.LevelHandler(log15.LvlDebug, logger.StreamHandler(os.Stdout, logger.TerminalFormatHandler(false, true))))
	AppLog.SetHandler(logger.LevelHandler(log15.LvlDebug, logger.StreamHandler(os.Stdout, logger.TerminalFormatHandler(false, true))))
	initLoggers()
	OnAppStart(initLoggers, -1)
}
func initLoggers() {
	trace, info, warn, err, app := logger.GetHandlers(BasePath, Config)
	// Set all the log handlers
	SetLog(oldLog, logger.MultiHandler(trace , info, warn, err))
	SetAppLog(AppLog, app)
}

// Set revel log
// DEPRECATED
func SetLog(mainLogger logger.MultiLogger, handler logger.LogHandler) {
	oldLog = mainLogger
	TRACE = logger.GetLogger("trace", oldLog)
	INFO = logger.GetLogger("info", oldLog)
	WARN = logger.GetLogger("warn", oldLog)
	ERROR = logger.GetLogger("error", oldLog)

	// Set log handlers
	SetLogHandlers(handler)
}

// Set handlers for all the TRACE, WARN, INFO, ERROR loggers, allows you to set custom handlers for all the loggers
// implement revel.LogHandler to set
// DEPRECATED
func SetLogHandlers(handler logger.LogHandler) {
	oldLog.SetHandler(logger.CallerFileHandler(handler))
}

// Set the application log and handler, if handler is nil it will
// use the same handler used to configure the application log before
func SetAppLog(appLog logger.MultiLogger, appHandler logger.LogHandler) {
	AppLog = appLog
	if appHandler==nil {
		SetAppLogHandlers(appLogHandler)
	} else {
		SetAppLogHandlers(appHandler)
	}
	// Set the handler for the default log output which may be the
	logger.SetDefaultLog(appLog)
}

// Set the handler for the application log
func SetAppLogHandlers(app logger.LogHandler) {
	appLogHandler = app
	AppLog.SetHandler(logger.CallerFileHandler(app))
}
