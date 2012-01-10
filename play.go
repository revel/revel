package play

import (
	"path"
	"path/filepath"
	"log"
	"os"
	"strings"
)

var (
	// App details
	AppName    string  // e.g. "sample"
	BasePath   string  // e.g. "/Users/robfig/gocode/src/play/sample"
	AppPath    string  // e.g. "/Users/robfig/gocode/src/play/sample/app"
	ViewsPath  string  // e.g. "/Users/robfig/gocode/src/play/sample/app/views"
	ImportPath string  // e.g. "play/sample"

	// Play installation details
	PlayTemplatePath string = "/Users/robfig/code/gocode/src/play/app/views"

	LOG = log.New(os.Stdout, "", log.Ldate | log.Ltime | log.Lshortfile)

	// Private
	playInit bool = false
)

func Init(basePath string) {
	AppName = filepath.Base(basePath)
	BasePath = basePath
	AppPath = path.Join(BasePath, "app")
	ViewsPath = path.Join(AppPath, "views")
	ImportPath = getImportPath(basePath)

	playInit = true
}

func CheckInit() {
	if ! playInit {
		panic("Play has not been initialized!")
	}
}

// The Import Path is how we can import its code.
// For example, the sample app resides in src/play/sample, and it must be
// imported as "play/sample/...".  Here, the import path is "play/sample".
// This assumes that the user's app is in a GOPATH, which requires the root of
// the packages to be "src".
func getImportPath(path string) string {
	srcIndex := strings.Index(path, "src")
	if srcIndex == -1 {
		LOG.Fatalf("App directory (%s) does not appear to be below \"src\". " +
			" I don't know how to import your code.  Please use GOPATH layout.",
			path)
	}
	return path[srcIndex+4:]
}
