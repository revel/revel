package rev

import (
	"go/build"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	// App details
	AppName    string // e.g. "sample"
	BasePath   string // e.g. "/Users/robfig/gocode/src/revel/sample"
	AppPath    string // e.g. "/Users/robfig/gocode/src/revel/sample/app"
	ViewsPath  string // e.g. "/Users/robfig/gocode/src/revel/sample/app/views"
	ImportPath string // e.g. "revel/sample"

	Config  *MergedConfig
	RunMode string // Application-defined (by default, "dev" or "prod")

	// Revel installation details
	RevelPath         string // e.g. "/Users/robfig/gocode/src/revel"
	RevelTemplatePath string // e.g. "/Users/robfig/gocode/src/revel/templates"

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

func Init(importPath string, mode string) {
	RunMode = mode

	// Find the user's app path.
	importPath = strings.TrimRight(importPath, "/")
	pkg, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln("Failed to import", importPath, "with error:", err)
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
	RevelPath = revelPkg.Dir
	RevelTemplatePath = path.Join(RevelPath, "templates")

	// Load application.conf
	Config, err = LoadConfig(path.Join(BasePath, "conf", "app.conf"))
	if err != nil {
		log.Fatalln("Failed to load app.conf:", err)
	}
	// Ensure that the selected runmode appears in app.conf.
	if !Config.HasSection(mode) {
		log.Fatalln("app.conf: No mode found:", mode)
	}
	Config.SetSection(mode)
	secretStr := Config.StringDefault("app.secret", "")
	if secretStr == "" {
		log.Fatalln("No app.secret provided.")
	}
	secretKey = []byte(secretStr)

	// Configure logging.
	TRACE = getLogger("trace")
	INFO = getLogger("info")
	WARN = getLogger("warn")
	ERROR = getLogger("error")

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

func CheckInit() {
	if !revelInit {
		panic("Revel has not been initialized!")
	}
}
