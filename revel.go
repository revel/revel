package revel

import (
	"github.com/agtorre/gocolorize"
	"github.com/robfig/config"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	REVEL_IMPORT_PATH = "github.com/robfig/revel"
)

type revelLogs struct {
	c gocolorize.Colorize
	w io.Writer
}

func (r *revelLogs) Write(p []byte) (n int, err error) {
	return r.w.Write([]byte(r.c.Paint(string(p))))
}

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
	DevMode bool   // if true, RunMode is a development mode.

	// Revel installation details
	RevelPath string // e.g. "/Users/robfig/gocode/src/revel"

	// Where to look for templates and configuration.
	// Ordered by priority.  (Earlier paths take precedence over later paths.)
	CodePaths     []string
	ConfPaths     []string
	TemplatePaths []string

	Modules []Module

	// Server config.
	//
	// Alert: This is how the app is configured, which may be different from
	// the current process reality.  For example, if the app is configured for
	// port 9000, HttpPort will always be 9000, even though in dev mode it is
	// run on a random port and proxied.
	HttpPort    int    // e.g. 9000
	HttpAddr    string // e.g. "", "127.0.0.1"
	HttpSsl     bool   // e.g. true if using ssl
	HttpSslCert string // e.g. "/path/to/cert.pem"
	HttpSslKey  string // e.g. "/path/to/key.pem"

	// All cookies dropped by the framework begin with this prefix.
	CookiePrefix string

	// Cookie flags
	CookieHttpOnly bool
	CookieSecure   bool

	// Delimiters to use when rendering templates
	TemplateDelims string

	//Logger colors
	colors = map[string]gocolorize.Colorize{
		"trace": gocolorize.NewColor("magenta"),
		"info":  gocolorize.NewColor("green"),
		"warn":  gocolorize.NewColor("yellow"),
		"error": gocolorize.NewColor("red"),
	}

	error_log = revelLogs{c: colors["error"], w: os.Stderr}

	// Loggers
	TRACE = log.New(ioutil.Discard, "TRACE ", log.Ldate|log.Ltime|log.Lshortfile)
	INFO  = log.New(ioutil.Discard, "INFO  ", log.Ldate|log.Ltime|log.Lshortfile)
	WARN  = log.New(ioutil.Discard, "WARN  ", log.Ldate|log.Ltime|log.Lshortfile)
	ERROR = log.New(&error_log, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile)

	Initialized bool

	// Private
	secretKey []byte // Key used to sign cookies. An empty key disables signing.
	packaged  bool   // If true, this is running from a pre-built package.
)

func init() {
	log.SetFlags(INFO.Flags())
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
	SourcePath = srcPath
	RunMode = mode

	if runtime.GOOS == "windows" {
		gocolorize.SetPlain(true)
	}

	// If the SourcePath is not specified, find it using build.Import.
	var revelSourcePath string // may be different from the app source path
	if SourcePath == "" {
		revelSourcePath, SourcePath = findSrcPaths(importPath)
	} else {
		// If the SourcePath was specified, assume both Revel and the app are within it.
		SourcePath = path.Clean(SourcePath)
		revelSourcePath = SourcePath
		packaged = true
	}

	RevelPath = path.Join(revelSourcePath, filepath.FromSlash(REVEL_IMPORT_PATH))
	BasePath = path.Join(SourcePath, filepath.FromSlash(importPath))
	AppPath = path.Join(BasePath, "app")
	ViewsPath = path.Join(AppPath, "views")

	CodePaths = []string{AppPath}

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

	// Configure properties from app.conf
	DevMode = Config.BoolDefault("mode.dev", false)
	HttpPort = Config.IntDefault("http.port", 9000)
	HttpAddr = Config.StringDefault("http.addr", "")
	HttpSsl = Config.BoolDefault("http.ssl", false)
	HttpSslCert = Config.StringDefault("http.sslcert", "")
	HttpSslKey = Config.StringDefault("http.sslkey", "")
	if HttpSsl {
		if HttpSslCert == "" {
			log.Fatalln("No http.sslcert provided.")
		}
		if HttpSslKey == "" {
			log.Fatalln("No http.sslkey provided.")
		}
	}

	AppName = Config.StringDefault("app.name", "(not set)")
	CookiePrefix = Config.StringDefault("cookie.prefix", "REVEL")
	CookieHttpOnly = Config.BoolDefault("cookie.httponly", false)
	CookieSecure = Config.BoolDefault("cookie.secure", false)
	TemplateDelims = Config.StringDefault("template.delimiters", "")
	if secretStr := Config.StringDefault("app.secret", ""); secretStr != "" {
		secretKey = []byte(secretStr)
	}

	// Configure logging.
	TRACE = getLogger("trace")
	INFO = getLogger("info")
	WARN = getLogger("warn")
	ERROR = getLogger("error")

	loadModules()

	Initialized = true
}

// Create a logger using log.* directives in app.conf plus the current settings
// on the default logger.
func getLogger(name string) *log.Logger {
	var logger *log.Logger

	// Create a logger with the requested output. (default to stderr)
	output := Config.StringDefault("log."+name+".output", "stderr")
	var newlog revelLogs

	switch output {
	case "stdout":
		newlog = revelLogs{c: colors[name], w: os.Stdout}
		logger = newLogger(&newlog)
	case "stderr":
		newlog = revelLogs{c: colors[name], w: os.Stderr}
		logger = newLogger(&newlog)
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
	return log.New(wr, "", INFO.Flags())
}

// findSrcPaths uses the "go/build" package to find the source root for Revel
// and the app.
func findSrcPaths(importPath string) (revelSourcePath, appSourcePath string) {
	var (
		gopaths = filepath.SplitList(build.Default.GOPATH)
		goroot  = build.Default.GOROOT
	)

	if len(gopaths) == 0 {
		ERROR.Fatalln("GOPATH environment variable is not set. ",
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.")
	}

	if ContainsString(gopaths, goroot) {
		ERROR.Fatalf("GOPATH (%s) must not include your GOROOT (%s). "+
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.",
			gopaths, goroot)
	}

	appPkg, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		ERROR.Fatalln("Failed to import", importPath, "with error:", err)
	}

	revelPkg, err := build.Import(REVEL_IMPORT_PATH, "", build.FindOnly)
	if err != nil {
		ERROR.Fatalln("Failed to find Revel with error:", err)
	}

	return revelPkg.SrcRoot, appPkg.SrcRoot
}

type Module struct {
	Name, ImportPath, Path string
}

func loadModules() {
	for _, key := range Config.Options("module.") {
		moduleImportPath := Config.StringDefault(key, "")
		if moduleImportPath == "" {
			continue
		}

		modulePath, err := ResolveImportPath(moduleImportPath)
		if err != nil {
			log.Fatalln("Failed to load module.  Import of", moduleImportPath, "failed:", err)
		}
		addModule(key[len("module."):], moduleImportPath, modulePath)
	}
}

// ResolveImportPath returns the filesystem path for the given import path.
// Returns an error if the import path could not be found.
func ResolveImportPath(importPath string) (string, error) {
	if packaged {
		return path.Join(SourcePath, importPath), nil
	}

	modPkg, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		return "", err
	}
	return modPkg.Dir, nil
}

func addModule(name, importPath, modulePath string) {
	Modules = append(Modules, Module{Name: name, ImportPath: importPath, Path: modulePath})
	if codePath := path.Join(modulePath, "app"); DirExists(codePath) {
		CodePaths = append(CodePaths, codePath)
		if viewsPath := path.Join(modulePath, "app", "views"); DirExists(viewsPath) {
			TemplatePaths = append(TemplatePaths, viewsPath)
		}
	}

	INFO.Print("Loaded module ", path.Base(modulePath))

	// Hack: There is presently no way for the testrunner module to add the
	// "test" subdirectory to the CodePaths.  So this does it instead.
	if importPath == "github.com/robfig/revel/modules/testrunner" {
		CodePaths = append(CodePaths, path.Join(BasePath, "tests"))
	}
}

// ModuleByName returns the module of the given name, if loaded.
func ModuleByName(name string) (m Module, found bool) {
	for _, module := range Modules {
		if module.Name == name {
			return module, true
		}
	}
	return Module{}, false
}

func CheckInit() {
	if !Initialized {
		panic("Revel has not been initialized!")
	}
}
