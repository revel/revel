package logger

import (
	"gopkg.in/inconshreveable/log15.v2"
	"log"
)
// The LogHanlder defines the interface to handle the log records
type (
// The Multilogger reduces the number of exposed defined logging variables,
// and allows the output to be easily refined
MultiLogger interface {
	log15.Logger
}
	LogHandler interface {
	log15.Handler
}
LogFormat interface {
	log15.Format
}

LogLevel log15.Lvl
)

// Set the systems default logger so if needed
func SetDefaultLog(fromLog MultiLogger) {
	log.SetOutput(loggerRewrite{Logger: fromLog, Level: log15.LvlInfo})
}