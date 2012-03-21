package play

import (
	"go/build"
	"log"
	"os"
	"path"
	"path/filepath"
)

var (
	// App details
	AppName    string // e.g. "sample"
	BasePath   string // e.g. "/Users/robfig/gocode/src/play/sample"
	AppPath    string // e.g. "/Users/robfig/gocode/src/play/sample/app"
	ViewsPath  string // e.g. "/Users/robfig/gocode/src/play/sample/app/views"
	ImportPath string // e.g. "play/sample"

	// Play installation details
	PlayTemplatePath string // e.g. "/Users/robfig/gocode/src/play/app/views"

	LOG = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	// Private
	playInit  bool   = false
	secretKey []byte = []byte("secret")
)

func Init(importPath string) {
	// Find the user's app path.
	pkg, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalf("Failed to import", importPath, "with error", err)
	}
	BasePath = pkg.Dir
	if BasePath == "" {
		log.Fatalf("Failed to find code.  Did you pass the import path?")
	}
	AppName = filepath.Base(BasePath)
	AppPath = path.Join(BasePath, "app")
	ViewsPath = path.Join(AppPath, "views")
	ImportPath = importPath

	// Find the provided resources.
	playPkg, err := build.Import("play", "", build.FindOnly)
	if err != nil {
		log.Fatalf("Failed to find play code.")
	}
	PlayTemplatePath = path.Join(playPkg.Dir, "app", "views")

	playInit = true
}

func CheckInit() {
	if !playInit {
		panic("Play has not been initialized!")
	}
}
