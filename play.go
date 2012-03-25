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
	Config     *MergedConfig

	// Play installation details
	PlayTemplatePath string // e.g. "/Users/robfig/gocode/src/play/app/views"

	LOG = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	// Private
	playInit  bool = false
	secretKey []byte
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

	// Load application.conf
	Config, err = LoadConfig(
		path.Join(BasePath, "conf", "app.conf"))
	if err != nil {
		log.Fatalln("Failed to load app.conf:", err)
	}
	secretStr, err := Config.String("app.secret")
	if err != nil {
		log.Fatalln("No app.secret provided.")
	}
	secretKey = []byte(secretStr)

	playInit = true
}

func CheckInit() {
	if !playInit {
		panic("Play has not been initialized!")
	}
}
