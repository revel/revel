package rev

import (
	"github.com/robfig/goconfig/config"
	"go/build"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	REVEL_IMPORT_PATH = "github.com/robfig/revel"
)

var (
	// App details
	AppName    string // e.g. "sample"
	BasePath   string // e.g. "/Users/robfig/gocode/src/corp/sample"
	AppPath    string // e.g. "/Users/robfig/gocode/src/corp/sample/app"
	ViewsPath  string // e.g. "/Users/robfig/gocode/src/corp/sample/app/views"
	ImportPath string // e.g. "corp/sample"
	SourcePath string // e.g. "/Users/robfig/gocode/src"

	Config  *MergedConfig
	RunMode string // Application-defined (by default, "dev" or "prod")

	// Revel installation details
	RevelPath string // e.g. "/Users/robfig/gocode/src/revel"

	// Where to look for templates and configuration.
	// Ordered by priority.  (Earlier paths take precedence over later paths.)
	ConfPaths     []string
	TemplatePaths []string

	ModulePaths []string

	DEFAULT = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)
	TRACE   = DEFAULT
	INFO    = DEFAULT
	WARN    = DEFAULT
	ERROR   = DEFAULT

	// Revel runs every function in this array after init.
	InitHooks []func()

	// Private
	revelInit bool
	secretKey []byte
)

func init() {
	log.SetFlags(DEFAULT.Flags())
}

// Init initializes Revel -- it provides paths for getting around the app.
//
// Params:
//   mode - the run mode, which determines which app.conf settings are used.
//   importPath - the Go import path of the application.
//   srcPath - the path to the source directory, containing Revel and the app.
//     If not specified (""), then a functioning Go installation is required.
func Init(mode, importPath, srcPath string) {
	// Ignore trailing slashes.
	ImportPath = strings.TrimRight(importPath, "/")
	SourcePath = strings.TrimRight(srcPath, "/")
	RunMode = mode

	if SourcePath == "" {
		SourcePath = findSrcPath(importPath)
	}

	RevelPath = path.Join(SourcePath, filepath.FromSlash(REVEL_IMPORT_PATH))
	BasePath = path.Join(SourcePath, filepath.FromSlash(importPath))
	AppPath = path.Join(BasePath, "app")
	ViewsPath = path.Join(AppPath, "views")

	ConfPaths = []string{
		path.Join(BasePath, "conf"),
		path.Join(RevelPath, "conf"),
	}

	TemplatePaths = []string{
		ViewsPath,
		path.Join(RevelPath, "templates"),
	}

	// Load app.conf
	var err error
	Config, err = LoadConfig("app.conf")
	if err != nil || Config == nil {
		log.Fatalln("Failed to load app.conf:", err)
	}
	// Ensure that the selected runmode appears in app.conf.
	// If empty string is passed as the mode, treat it as "DEFAULT"
	if mode == "" {
		mode = config.DEFAULT_SECTION
	}
	if !Config.HasSection(mode) {
		log.Fatalln("app.conf: No mode found:", mode)
	}
	Config.SetSection(mode)
	secretStr := Config.StringDefault("app.secret", "")
	if secretStr == "" {
		log.Fatalln("No app.secret provided.")
	}
	secretKey = []byte(secretStr)

	AppName = Config.StringDefault("app.name", "(not set)")

	// Configure logging.
	TRACE = getLogger("trace")
	INFO = getLogger("info")
	WARN = getLogger("warn")
	ERROR = getLogger("error")

	loadModules()

	for _, hook := range InitHooks {
		hook()
	}

	revelInit = true
}

// Create a logger using log.* directives in app.conf plus the current settings
// on the default logger.
func getLogger(name string) *log.Logger {
	var logger *log.Logger

	// Create a logger with the requested output. (default to stderr)
	output := Config.StringDefault("log."+name+".output", "stderr")

	switch output {
	case "stdout":
		logger = newLogger(os.Stdout)
	case "stderr":
		logger = newLogger(os.Stderr)
	default:
		if output == "off" {
			output = os.DevNull
		}

		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalln("Failed to open log file", output, ":", err)
		}
		logger = newLogger(file)
	}

	// Set the prefix / flags.
	flags, found := Config.Int("log." + name + ".flags")
	if found {
		logger.SetFlags(flags)
	}

	prefix, found := Config.String("log." + name + ".prefix")
	if found {
		logger.SetPrefix(prefix)
	}

	return logger
}

func newLogger(wr io.Writer) *log.Logger {
	return log.New(wr, DEFAULT.Prefix(), DEFAULT.Flags())
}

// findSrcPath uses the "go/build" package to find the source root.
func findSrcPath(importPath string) string {
	appPkg, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln("Failed to import", importPath, "with error:", err)
	}

	revelPkg, err := build.Import(REVEL_IMPORT_PATH, "", build.FindOnly)
	if err != nil {
		log.Fatalln("Failed to find Revel with error:", err)
	}

	// Find the source path from each, and ensure they are equal.
	if revelPkg.SrcRoot != appPkg.SrcRoot {
		log.Fatalln("Revel must be installed in the same GOPATH as your app."+
			"\nRevel source root:", revelPkg.SrcRoot,
			"\nApp source root:", appPkg.SrcRoot)
	}

	return appPkg.SrcRoot
}

func loadModules() {
	for _, root := range []string{RevelPath, BasePath} {
		moduleDirPath := path.Join(root, "modules")
		moduleDir, err := os.Open(moduleDirPath)
		if err != nil {
			if !os.IsNotExist(err) {
				ERROR.Print("Error stat'ing path ", moduleDirPath, ":", err)
			}
			continue
		}
		defer moduleDir.Close()

		names, err := moduleDir.Readdirnames(0)
		if err != nil {
			ERROR.Print("Error listing dir ", moduleDirPath, ":", err)
			continue
		}

		for _, moduleName := range names {
			addModule(path.Join(moduleDirPath, moduleName))
		}
	}
}

func addModule(modulePath string) {
	ModulePaths = append(ModulePaths, modulePath)
	if viewsPath := path.Join(modulePath, "app", "views"); DirExists(viewsPath) {
		TemplatePaths = append(TemplatePaths, viewsPath)
	}
	INFO.Print("Loaded module ", path.Base(modulePath))
}

func CheckInit() {
	if !revelInit {
		panic("Revel has not been initialized!")
	}
}
