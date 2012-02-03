package play

import (
	"path"
	"path/filepath"
	"log"
	"os"
)

var (
	// App details
	AppName    string  // e.g. "sample"
	BasePath   string  // e.g. "/Users/robfig/gocode/src/play/sample"
	AppPath    string  // e.g. "/Users/robfig/gocode/src/play/sample/app"
	ViewsPath  string  // e.g. "/Users/robfig/gocode/src/play/sample/app/views"
	ImportPath string  // e.g. "play/sample"

	// Play installation details
	PlayTemplatePath string  // e.g. "/Users/robfig/gocode/src/play/app/views"

	LOG = log.New(os.Stdout, "", log.Ldate | log.Ltime | log.Lshortfile)

	// Private
	playInit bool = false
)

func Init(importPath string) {
	BasePath = FindSource(importPath)
	if BasePath == "" {
		log.Fatalf("Failed to find code.  Did you pass the import path?")
	}
	AppName = filepath.Base(BasePath)
	AppPath = path.Join(BasePath, "app")
	ViewsPath = path.Join(AppPath, "views")
	ImportPath = importPath
	PlayTemplatePath = FindSource("play/app/views")

	playInit = true
}

func CheckInit() {
	if ! playInit {
		panic("Play has not been initialized!")
	}
}
