package play

import (
	"path"
	"log"
//	l4g "log4go.googlecode.com/hg"
	"os"
)

var AppName = "sample"
var BasePath = "/Users/robfig/code/gocode/src/play/sample"
var AppPath = path.Join(BasePath, "app")
// TODO: Get log4go to work and use that instead.
//var LOG = l4g.NewDefaultLogger(l4g.FINEST)
var LOG = log.New(os.Stdout, "", log.Ldate | log.Ltime | log.Lshortfile)

var PlayPath = "/Users/robfig/code/gocode/src/play"
var BaseImportPath = "play"
