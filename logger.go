package revel

import (
	"github.com/revel/revel/logger"
	"github.com/revel/log15"
	"log"
	"os"
)

//Logger
var (
	// The root log is what all other logs are branched from, meaning if you set the handler for the root
	// it will adjust all children
	RootLog = logger.New()
	// This logger is the application logger, use this for your application log messages - ie jobs and startup,
	// Use Controller.Log for Controller logging
	// The requests are logged to this logger with the context of `section:requestlog`
	AppLog = RootLog.New("module","app")
	// This is the logger revel writes to, added log messages will have a context of module:revel in them
	// It is based off of `RootLog`
	RevelLog = RootLog.New("module", "revel")

	// This is the handler for the AppLog, it is stored so that if the AppLog is changed it can be assigned to the
	// new AppLog
	appLogHandler *logger.CompositeMultiHandler

	// This oldLog is the revel logger, historical for revel, The application should use the AppLog or the Controller.oldLog
	// DEPRECATED
	oldLog = AppLog.New("section", "deprecated")
	// System logger
	SysLog = AppLog.New("section", "system")
	// DEPRECATED Use AppLog
	TRACE = log.New(os.Stderr, "TRACE ", log.Ldate|log.Ltime|log.Lshortfile)
	// DEPRECATED Use AppLog
	INFO = log.New(os.Stderr, "INFO ", log.Ldate|log.Ltime|log.Lshortfile)
	// DEPRECATED Use AppLog
	WARN = log.New(os.Stderr, "WARN ", log.Ldate|log.Ltime|log.Lshortfile)
	// DEPRECATED Use AppLog
	ERROR = log.New(os.Stderr, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile)
)

func init() {

	RootLog.SetHandler(logger.LevelHandler(logger.LogLevel(log15.LvlDebug), logger.StreamHandler(os.Stdout, logger.TerminalFormatHandler(false, true))))
	initLoggers()
	OnAppStart(initLoggers, -1)

}
func initLoggers() {
	appHandle := logger.InitializeFromConfig(BasePath, Config)

	// Set all the log handlers
	setLog(oldLog, appHandle)
	setAppLog(AppLog, appHandle)
}

// Set handlers for the old logs, ERROR,TRACE,WARN,INFO
// DEPRECATED
func setLog(mainLogger logger.MultiLogger, handler logger.LogHandler) {
	oldLog = mainLogger
	TRACE = logger.GetLogger("trace", oldLog)
	INFO = logger.GetLogger("info", oldLog)
	WARN = logger.GetLogger("warn", oldLog)
	ERROR = logger.GetLogger("error", oldLog)

	// This code sets the caller context for the old package
	// since the skip level is different
	oldLog.SetStackDepth(5)
	oldLog.SetHandler(handler)
}

// Set the application log and handler, if handler is nil it will
// use the same handler used to configure the application log before
func setAppLog(appLog logger.MultiLogger, appHandler *logger.CompositeMultiHandler) {
	if appLog != nil {
		AppLog = appLog
	}
	if appHandler != nil {
		appLogHandler = appHandler
		// Set the app log and the handler for all forked loggers
		RootLog.SetHandler(appLogHandler)

		// Set the system log handler - this sets golang writer stream to the
		// sysLog router
		logger.SetDefaultLog(SysLog)
		SysLog.SetStackDepth(5)
		SysLog.SetHandler(appLogHandler)
	}
}

// Return the root log handler
func GetRootLogHandler() *logger.CompositeMultiHandler {
	return appLogHandler
}
