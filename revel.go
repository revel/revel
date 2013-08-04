package revel

import (
	"flag"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/robfig/config"
	"github.com/robfig/humanize"
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

	CookiePrefix string // All cookies dropped by the framework begin with this prefix.
	LogToStderr  bool   // If true, hard code logging configuration to logtostderr

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

	RevelPath = filepath.Join(revelSourcePath, filepath.FromSlash(REVEL_IMPORT_PATH))
	BasePath = filepath.Join(SourcePath, filepath.FromSlash(importPath))
	AppPath = filepath.Join(BasePath, "app")
	ViewsPath = filepath.Join(AppPath, "views")

	CodePaths = []string{AppPath}

	ConfPaths = []string{
		filepath.Join(BasePath, "conf"),
		filepath.Join(RevelPath, "conf"),
	}

	TemplatePaths = []string{
		ViewsPath,
		filepath.Join(RevelPath, "templates"),
	}

	// Load app.conf
	var err error
	Config, err = LoadConfig("app.conf")
	if err != nil || Config == nil {
		glog.Fatalln("Failed to load app.conf:", err)
	}
	// Ensure that the selected runmode appears in app.conf.
	// If empty string is passed as the mode, treat it as "DEFAULT"
	if mode == "" {
		mode = config.DEFAULT_SECTION
	}
	if !Config.HasSection(mode) {
		glog.Fatalln("app.conf: No mode found:", mode)
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
	if secretStr := Config.StringDefault("app.secret", ""); secretStr != "" {
		secretKey = []byte(secretStr)
	}

	Initialized = true
}

// ConfigureLogging applies the configuration in revel.Config to the glog flags.
// Logger flags specified explicitly on the command line are not changed.
func ConfigureLogging() {
	// Get the flags specified on the command line.
	specifiedFlags := make(map[string]struct{})
	flag.Visit(func(f *flag.Flag) { specifiedFlags[f.Name] = struct{}{} })

	// For each logger option in app.conf..
	var err error
	for _, option := range Config.Options("log.") {
		val, _ := Config.String(option)
		switch flagname := option[len("log."):]; flagname {
		case "v", "vmodule", "logtostderr", "alsologtostderr", "stderrthreshold", "log_dir":
			// If it was specified on the command line, don't set it from app.conf
			if _, ok := specifiedFlags[flagname]; ok {
				continue
			}

			// Look up the flag and set it.
			// If it's log_dir, make it into an absolute path and creat it if necessary.
			if flagname == "log_dir" {
				if val, err = filepath.Abs(val); err != nil {
					glog.Fatalln("Failed to get absolute path to log_dir:", err)
				}
				os.MkdirAll(val, 0777) // Create the log dir if it doesn't already exist.
			}
			if err = flag.Set(flagname, val); err != nil {
				glog.Fatalf("Failed to set glog option for %s=%s: %s", flagname, val, err)
			}
		case "maxsize":
			if glog.MaxSize, err = humanize.ParseBytes(val); err != nil {
				glog.Fatalf("Failed to parse log.MaxSize=%s: %s", val, err)
			}
		}
	}
}

// findSrcPaths uses the "go/build" package to find the source root for Revel
// and the app.
func findSrcPaths(importPath string) (revelSourcePath, appSourcePath string) {
	var (
		gopaths = filepath.SplitList(build.Default.GOPATH)
		goroot  = build.Default.GOROOT
	)

	if len(gopaths) == 0 {
		glog.Fatal("GOPATH environment variable is not set. ",
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.")
	}

	if ContainsString(gopaths, goroot) {
		glog.Fatalf("GOPATH (%s) must not include your GOROOT (%s). "+
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.",
			gopaths, goroot)
	}

	appPkg, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		glog.Fatalln("Failed to import", importPath, "with error:", err)
	}

	revelPkg, err := build.Import(REVEL_IMPORT_PATH, "", build.FindOnly)
	if err != nil {
		glog.Fatalln("Failed to find Revel with error:", err)
	}

	return revelPkg.SrcRoot, appPkg.SrcRoot
}

type Module struct {
	Name, ImportPath, Path string
}

// LoadModules looks through Config for all modules that need to be loaded,
// adding their controllers and templates to the revel application.
func LoadModules() {
	for _, key := range Config.Options("module.") {
		moduleImportPath := Config.StringDefault(key, "")
		if moduleImportPath == "" {
			continue
		}

		modulePath, err := ResolveImportPath(moduleImportPath)
		if err != nil {
			glog.Fatalln("Failed to load module.  Import of", moduleImportPath, "failed:", err)
		}
		addModule(key[len("module."):], moduleImportPath, modulePath)
	}
}

// ResolveImportPath returns the filesystem path for the given import path.
// Returns an error if the import path could not be found.
func ResolveImportPath(importPath string) (string, error) {
	if packaged {
		return filepath.Join(SourcePath, filepath.FromSlash(importPath)), nil
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
	glog.Info("Loaded module ", filepath.Base(modulePath))

	// Hack: There is presently no way for the testrunner module to add the
	// "test" subdirectory to the CodePaths.  So this does it instead.
	if importPath == "github.com/robfig/revel/modules/testrunner" {
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

func CheckInit() {
	if !Initialized {
		panic("Revel has not been initialized!")
	}
}
