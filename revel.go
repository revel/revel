package rev

import (
	"go/build"
	"log"
	"os"
	"path"
	"path/filepath"
)

type RunMode string

const (
	DEV  RunMode = "dev"
	PROD         = "prod"
)

var (
	// App details
	AppName    string // e.g. "sample"
	BasePath   string // e.g. "/Users/robfig/gocode/src/revel/sample"
	AppPath    string // e.g. "/Users/robfig/gocode/src/revel/sample/app"
	ViewsPath  string // e.g. "/Users/robfig/gocode/src/revel/sample/app/views"
	ImportPath string // e.g. "revel/sample"

	Config  *MergedConfig
	AppMode RunMode // DEV or PROD

	// Revel installation details
	RevelTemplatePath string // e.g. "/Users/robfig/gocode/src/revel/templates"

	LOG = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	// Private
	revelInit bool = false
	secretKey []byte
)

func Init(importPath string, mode RunMode) {
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
	revelPkg, err := build.Import("github.com/robfig/revel", "", build.FindOnly)
	if err != nil {
		log.Fatalf("Failed to find revel code.")
	}
	RevelTemplatePath = path.Join(revelPkg.Dir, "templates")

	// Load application.conf
	Config, err = LoadConfig(
		path.Join(BasePath, "conf", "app.conf"))
	if err != nil {
		log.Fatalln("Failed to load app.conf:", err)
	}
	Config.SetSection(string(mode))
	secretStr, err := Config.String("app.secret")
	if err != nil {
		log.Fatalln("No app.secret provided.")
	}
	secretKey = []byte(secretStr)

	revelInit = true
}

func CheckInit() {
	if !revelInit {
		panic("Revel has not been initialized!")
	}
}
