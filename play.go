package play

import (
	"path"
	"log"
//	l4g "log4go.googlecode.com/hg"
	"os"
)

// TODO: Get log4go to work and use that instead.
//var LOG = l4g.NewDefaultLogger(l4g.FINEST)
var LOG = log.New(os.Stdout, "", log.Ldate | log.Ltime | log.Lshortfile)

// App details (to eventually come from application.conf)
var AppName = "sample"
var BasePath = "/Users/robfig/code/gocode/src/play/sample"
var AppPath = path.Join(BasePath, "app")
var ViewsPath = path.Join(AppPath, "views")

// Play installation details
var PlayPath = "/Users/robfig/code/gocode/src/play"
var BaseImportPath = "play"
var PlayTemplatePath = "/Users/robfig/code/gocode/src/play/app/views"
