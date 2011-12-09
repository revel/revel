package play

import (
	"path"
	"log"
//	l4g "log4go.googlecode.com/hg"
	"os"
)

var BasePath = "/Users/robfig/code/goplay/sample"
var AppPath = path.Join(BasePath, "app")
// TODO: Get log4go to work and use that instead.
//var LOG = l4g.NewDefaultLogger(l4g.FINEST)
var LOG = log.New(os.Stdout, "", log.Ldate | log.Ltime | log.Lshortfile)

