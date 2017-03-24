// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/agtorre/gocolorize"
	"github.com/revel/config"
)

const (
	// RevelImportPath Revel framework import path
	RevelImportPath = "github.com/revel/revel"
)

type revelLogs struct {
	c gocolorize.Colorize
	w io.Writer
}

func (r *revelLogs) Write(p []byte) (n int, err error) {
	return r.w.Write([]byte(r.c.Paint(string(p))))
}

// App details
var (
	AppName    string // e.g. "sample"
	AppRoot    string // e.g. "/app1"
	BasePath   string // e.g. "$GOPATH/src/corp/sample"
	AppPath    string // e.g. "$GOPATH/src/corp/sample/app"
	ViewsPath  string // e.g. "$GOPATH/src/corp/sample/app/views"
	ImportPath string // e.g. "corp/sample"
	SourcePath string // e.g. "$GOPATH/src"

	Config  *config.Context
	RunMode string // Application-defined (by default, "dev" or "prod")
	DevMode bool   // if true, RunMode is a development mode.

	// Revel installation details
	RevelPath string // e.g. "$GOPATH/src/github.com/revel/revel"

	// Where to look for templates
	// Ordered by priority. (Earlier paths take precedence over later paths.)
	CodePaths     []string
	TemplatePaths []string

	// ConfPaths where to look for configurations
	// Config load order
	// 1. framework (revel/conf/*)
	// 2. application (conf/*)
	// 3. user supplied configs (...) - User configs can override/add any from above
	ConfPaths []string

	Modules []Module

	// Server config.
	//
	// Alert: This is how the app is configured, which may be different from
	// the current process reality.  For example, if the app is configured for
	// port 9000, HTTPPort will always be 9000, even though in dev mode it is
	// run on a random port and proxied.
	HTTPPort    int    // e.g. 9000
	HTTPAddr    string // e.g. "", "127.0.0.1"
	HTTPSsl     bool   // e.g. true if using ssl
	HTTPSslCert string // e.g. "/path/to/cert.pem"
	HTTPSslKey  string // e.g. "/path/to/key.pem"

	// All cookies dropped by the framework begin with this prefix.
	CookiePrefix string
	// Cookie domain
	CookieDomain string
	// Cookie flags
	CookieSecure bool

	// Delimiters to use when rendering templates
	TemplateDelims string

	//Logger colors
	colors = map[string]gocolorize.Colorize{
		"trace": gocolorize.NewColor("magenta"),
		"info":  gocolorize.NewColor("white"),
		"warn":  gocolorize.NewColor("yellow"),
		"error": gocolorize.NewColor("red"),
	}

	errorLog = revelLogs{c: colors["error"], w: os.Stderr}

	// Loggers
	TRACE = log.New(ioutil.Discard, "TRACE ", log.Ldate|log.Ltime|log.Lshortfile)
	INFO  = log.New(ioutil.Discard, "INFO ", log.Ldate|log.Ltime|log.Lshortfile)
	WARN  = log.New(ioutil.Discard, "WARN ", log.Ldate|log.Ltime|log.Lshortfile)
	ERROR = log.New(&errorLog, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile)

	// Revel request access log, not exposed from package.
	// However output settings can be controlled from app.conf
	requestLog           = log.New(ioutil.Discard, "", 0)
	requestLogTimeFormat = "2006/01/02 15:04:05.000"

	Initialized bool

	// Private
	secretKey []byte // Key used to sign cookies. An empty key disables signing.
	packaged  bool   // If true, this is running from a pre-built package.
)

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
		SourcePath = filepath.Clean(SourcePath)
		revelSourcePath = SourcePath
		packaged = true
	}

	RevelPath = filepath.Join(revelSourcePath, filepath.FromSlash(RevelImportPath))
	BasePath = filepath.Join(SourcePath, filepath.FromSlash(importPath))
	AppPath = filepath.Join(BasePath, "app")
	ViewsPath = filepath.Join(AppPath, "views")

	CodePaths = []string{AppPath}

	if ConfPaths == nil {
		ConfPaths = []string{}
	}

	// Config load order
	// 1. framework (revel/conf/*)
	// 2. application (conf/*)
	// 3. user supplied configs (...) - User configs can override/add any from above
	ConfPaths = append(
		[]string{
			filepath.Join(RevelPath, "conf"),
			filepath.Join(BasePath, "conf"),
		},
		ConfPaths...)

	TemplatePaths = []string{
		ViewsPath,
		filepath.Join(RevelPath, "templates"),
	}

	// Load app.conf
	var err error
	Config, err = config.LoadContext("app.conf", ConfPaths)
	if err != nil || Config == nil {
		log.Fatalln("Failed to load app.conf:", err)
	}
	// Ensure that the selected runmode appears in app.conf.
	// If empty string is passed as the mode, treat it as "DEFAULT"
	if mode == "" {
		mode = config.DefaultSection
	}
	if !Config.HasSection(mode) {
		log.Fatalln("app.conf: No mode found:", mode)
	}
	Config.SetSection(mode)

	// Configure properties from app.conf
	DevMode = Config.BoolDefault("mode.dev", false)
	HTTPPort = Config.IntDefault("http.port", 9000)
	HTTPAddr = Config.StringDefault("http.addr", "")
	HTTPSsl = Config.BoolDefault("http.ssl", false)
	HTTPSslCert = Config.StringDefault("http.sslcert", "")
	HTTPSslKey = Config.StringDefault("http.sslkey", "")
	if HTTPSsl {
		if HTTPSslCert == "" {
			log.Fatalln("No http.sslcert provided.")
		}
		if HTTPSslKey == "" {
			log.Fatalln("No http.sslkey provided.")
		}
	}

	AppName = Config.StringDefault("app.name", "(not set)")
	AppRoot = Config.StringDefault("app.root", "")
	CookiePrefix = Config.StringDefault("cookie.prefix", "REVEL")
	CookieDomain = Config.StringDefault("cookie.domain", "")
	CookieSecure = Config.BoolDefault("cookie.secure", HTTPSsl)
	TemplateDelims = Config.StringDefault("template.delimiters", "")
	if secretStr := Config.StringDefault("app.secret", ""); secretStr != "" {
		secretKey = []byte(secretStr)
	}

	// Configure logging
	if !Config.BoolDefault("log.colorize", true) {
		gocolorize.SetPlain(true)
	}

	TRACE = getLogger("trace")
	INFO = getLogger("info")
	WARN = getLogger("warn")
	ERROR = getLogger("error")

	// Revel request access logger, not exposed from package.
	// However output settings can be controlled from app.conf
	requestLog = getLogger("request")

	loadModules()

	Initialized = true
	INFO.Printf("Initialized Revel v%s (%s) for %s", Version, BuildDate, MinimumGoVersion)
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
	case "off":
		return newLogger(ioutil.Discard)
	default:
		if !filepath.IsAbs(output) {
			output = filepath.Join(BasePath, output)
		}

		logPath := filepath.Dir(output)
		if err := createDir(logPath); err != nil {
			log.Fatalln(err)
		}

		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalln("Failed to open log file", output, ":", err)
		}
		logger = newLogger(file)
	}

	if strings.EqualFold(name, "request") {
		logger.SetFlags(0)
		return logger
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

	revelPkg, err := build.Import(RevelImportPath, "", build.FindOnly)
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
		return filepath.Join(SourcePath, importPath), nil
	}

	modPkg, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		return "", err
	}
	return modPkg.Dir, nil
}

func addModule(name, importPath, modulePath string) {
	Modules = append(Modules, Module{Name: name, ImportPath: importPath, Path: modulePath})
	if codePath := filepath.Join(modulePath, "app"); DirExists(codePath) {
		CodePaths = append(CodePaths, codePath)
		if viewsPath := filepath.Join(modulePath, "app", "views"); DirExists(viewsPath) {
			TemplatePaths = append(TemplatePaths, viewsPath)
		}
	}

	INFO.Print("Loaded module ", filepath.Base(modulePath))

	// Hack: There is presently no way for the testrunner module to add the
	// "test" subdirectory to the CodePaths.  So this does it instead.
	if importPath == Config.StringDefault("module.testrunner", "github.com/revel/modules/testrunner") {
		CodePaths = append(CodePaths, filepath.Join(BasePath, "tests"))
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

// CheckInit method checks `revel.Initialized` if not initialized it panics
func CheckInit() {
	if !Initialized {
		panic("Revel has not been initialized!")
	}
}

func init() {
	log.SetFlags(INFO.Flags())
}
