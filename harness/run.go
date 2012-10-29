package harness

import (
	"bytes"
	"fmt"
	"github.com/robfig/revel"
	"go/build"
	"io"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"text/template"
)

var (
	cmd *exec.Cmd // The app server cmd

	importErrorPattern = regexp.MustCompile(
		"import \"([^\"]+)\": cannot find package")
)

// Run the Revel program, optionally using the harness.
// If the Harness is not used to manage the program, it is returned to the caller.
// (e.g. the return value is nil if useHarness is true, else it is the running program)
func StartApp(useHarness bool) *exec.Cmd {
	// If we are in prod mode, just build and run the application.
	if !useHarness {
		rev.INFO.Println("Building...")
		binName, err := Build()
		if err != nil {
			rev.ERROR.Fatalln(err)
		}
		start(binName, getAppAddress(), getAppPort())
		return cmd
	}

	// If the harness exits, be sure to kill the app server.
	defer func() {
		if cmd != nil {
			cmd.Process.Kill()
			cmd = nil
		}
	}()

	// Run a reverse proxy to it.
	harness := NewHarness()
	harness.Run()
	return nil
}

// Build the app:
// 1. Generate the the main.go file.
// 2. Run the appropriate "go build" command.
// Requires that rev.Init has been called previously.
// Returns the path to the built binary, and an error if there was a problem building it.
func Build() (binaryPath string, compileError *rev.Error) {
	sourceInfo, compileError := ProcessSource()
	if compileError != nil {
		return "", compileError
	}

	tmpl := template.New("RegisterControllers")
	tmpl = template.Must(tmpl.Parse(REGISTER_CONTROLLERS))
	var registerControllerSource string = rev.ExecuteTemplate(tmpl, map[string]interface{}{
		"Controllers":     sourceInfo.ControllerSpecs,
		"ValidationKeys":  sourceInfo.ValidationKeys,
		"ImportPaths":     uniqueImportPaths(sourceInfo),
		"UnitTests":       sourceInfo.UnitTests,
		"FunctionalTests": sourceInfo.FunctionalTests,
	})

	// Terminate the server if it's already running.
	if cmd != nil && (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) {
		rev.TRACE.Println("Killing revel server pid", cmd.Process.Pid)
		err := cmd.Process.Kill()
		if err != nil {
			rev.ERROR.Fatalln("Failed to kill revel server:", err)
		}
	}

	// Create a fresh temp dir.
	tmpPath := path.Join(rev.AppPath, "tmp")
	err := os.RemoveAll(tmpPath)
	if err != nil {
		rev.ERROR.Println("Failed to remove tmp dir:", err)
	}
	err = os.Mkdir(tmpPath, 0777)
	if err != nil {
		rev.ERROR.Fatalf("Failed to make tmp directory: %v", err)
	}

	// Create the new file
	controllersFile, err := os.Create(path.Join(tmpPath, "main.go"))
	defer controllersFile.Close()
	if err != nil {
		rev.ERROR.Fatalf("Failed to create main.go: %v", err)
	}
	_, err = controllersFile.WriteString(registerControllerSource)
	if err != nil {
		rev.ERROR.Fatalf("Failed to write to main.go: %v", err)
	}

	// Build the user program (all code under app).
	// It relies on the user having "go" installed.
	goPath, err := exec.LookPath("go")
	if err != nil {
		rev.ERROR.Fatalf("Go executable not found in PATH.")
	}

	ctx := build.Default
	pkg, err := ctx.Import(rev.ImportPath, "", build.FindOnly)
	if err != nil {
		rev.ERROR.Fatalln("Failure importing", rev.ImportPath)
	}
	binName := path.Join(pkg.BinDir, path.Base(rev.BasePath))
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	gotten := make(map[string]struct{})
	for {
		buildCmd := exec.Command(goPath, "build", "-o", binName, path.Join(rev.ImportPath, "app", "tmp"))
		rev.TRACE.Println("Exec:", buildCmd.Args)
		output, err := buildCmd.CombinedOutput()

		// If the build succeeded, we're done.
		if err == nil {
			return binName, nil
		}
		rev.TRACE.Println(string(output))

		// See if it was an import error that we can go get.
		matches := importErrorPattern.FindStringSubmatch(string(output))
		if matches == nil {
			return "", newCompileError(output)
		}

		// Ensure we haven't already tried to go get it.
		pkgName := matches[1]
		if _, alreadyTried := gotten[pkgName]; alreadyTried {
			return "", newCompileError(output)
		}
		gotten[pkgName] = struct{}{}

		// Execute "go get <pkg>"
		getCmd := exec.Command(goPath, "get", pkgName)
		rev.TRACE.Println("Exec:", getCmd.Args)
		getOutput, err := getCmd.CombinedOutput()
		if err != nil {
			rev.TRACE.Println(string(getOutput))
			return "", newCompileError(output)
		}

		// Success getting the import, attempt to build again.
	}
	rev.ERROR.Fatalf("Not reachable")
	return "", nil
}

// Start the application server, waiting until it has started up.
// Panics if startup fails.
func start(binName, addr string, port int) {
	// Run the server, via tmp/main.go.
	cmd = exec.Command(binName,
		fmt.Sprintf("-port=%d", port),
		fmt.Sprintf("-importPath=%s", rev.ImportPath),
		fmt.Sprintf("-runMode=%s", rev.RunMode),
	)
	rev.TRACE.Println("Exec app:", cmd.Path, cmd.Args)
	listeningWriter := startupListeningWriter{os.Stdout, make(chan bool)}
	cmd.Stdout = listeningWriter
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		rev.ERROR.Fatalln("Error running:", err)
	}

	<-listeningWriter.notifyReady
}

// A io.Writer that copies to the destination, and listens for "Listening on.."
// in the stream.  (Which tells us when the revel server has finished starting up)
// This is super ghetto, but by far the simplest thing that should work.
type startupListeningWriter struct {
	dest        io.Writer
	notifyReady chan bool
}

func (w startupListeningWriter) Write(p []byte) (n int, err error) {
	if w.notifyReady != nil && bytes.Contains(p, []byte("Listening")) {
		w.notifyReady <- true
		w.notifyReady = nil
	}
	return w.dest.Write(p)
}

// Return port that the app should listen on.
// 9000 by default.
func getAppPort() int {
	return rev.Config.IntDefault("http.port", 9000)
}

// Return address that the app should listen on.
// Wildcard by default.
func getAppAddress() string {
	return rev.Config.StringDefault("http.addr", "")
}

// Find an unused port
func getFreePort() (port int) {
	conn, err := net.Listen("tcp", ":0")
	if err != nil {
		rev.ERROR.Fatal(err)
	}

	port = conn.Addr().(*net.TCPAddr).Port
	err = conn.Close()
	if err != nil {
		rev.ERROR.Fatal(err)
	}
	return port
}

// Looks through all the method args and returns a set of unique import paths
// that cover all the method arg types.
func uniqueImportPaths(src *SourceInfo) (paths []string) {
	importPathMap := make(map[string]bool)
	typeArrays := [][]*TypeInfo{src.ControllerSpecs, src.UnitTests, src.FunctionalTests}
	for _, specs := range typeArrays {
		for _, spec := range specs {
			importPathMap[spec.ImportPath] = true
			for _, methSpec := range spec.MethodSpecs {
				for _, methArg := range methSpec.Args {
					if methArg.ImportPath != "" {
						importPathMap[methArg.ImportPath] = true
					}
				}
			}
		}
	}

	for importPath := range importPathMap {
		paths = append(paths, importPath)
	}

	return
}

// Parse the output of the "go build" command.
// Return a detailed Error.
func newCompileError(output []byte) *rev.Error {
	errorMatch := regexp.MustCompile(`(?m)^([^:#]+):(\d+):(\d+:)? (.*)$`).
		FindSubmatch(output)
	if errorMatch == nil {
		rev.ERROR.Println("Failed to parse build errors:\n", string(output))
		return &rev.Error{
			SourceType:  "Go code",
			Title:       "Go Compilation Error",
			Description: "See console for build error.",
		}
	}

	// Read the source for the offending file.
	var (
		relFilename    = string(errorMatch[1]) // e.g. "src/revel/sample/app/controllers/app.go"
		absFilename, _ = filepath.Abs(relFilename)
		line, _        = strconv.Atoi(string(errorMatch[2]))
		description    = string(errorMatch[4])
		compileError   = &rev.Error{
			SourceType:  "Go code",
			Title:       "Go Compilation Error",
			Path:        relFilename,
			Description: description,
			Line:        line,
		}
	)

	fileStr, err := rev.ReadLines(absFilename)
	if err != nil {
		compileError.MetaError = absFilename + ": " + err.Error()
		rev.ERROR.Println(compileError.MetaError)
		return compileError
	}

	compileError.SourceLines = fileStr
	return compileError
}

const REGISTER_CONTROLLERS = `package main

import (
	"flag"
	"reflect"
	"github.com/robfig/revel"{{range .ImportPaths}}
	"{{.}}"{{end}}
)

var (
	runMode    *string = flag.String("runMode", "", "Run mode.")
	port       *int    = flag.Int("port", 0, "By default, read from app.conf")
	importPath *string = flag.String("importPath", "", "Go Import Path for the app.")
	srcPath    *string = flag.String("srcPath", "", "Path to the source root.")

	// So compiler won't complain if the generated code doesn't reference reflect package...
	_ = reflect.Invalid
)

func main() {
	rev.INFO.Println("Running revel server")
	flag.Parse()
	rev.Init(*runMode, *importPath, *srcPath)
	{{range $i, $c := .Controllers}}
	rev.RegisterController((*{{.PackageName}}.{{.StructName}})(nil),
		[]*rev.MethodType{
			{{range .MethodSpecs}}&rev.MethodType{
				Name: "{{.Name}}",
				Args: []*rev.MethodArg{ {{range .Args}}
					&rev.MethodArg{Name: "{{.Name}}", Type: reflect.TypeOf((*{{.TypeName}})(nil)) },{{end}}
			  },
				RenderArgNames: map[int][]string{ {{range .RenderCalls}}
					{{.Line}}: []string{ {{range .Names}}
						"{{.}}",{{end}}
					},{{end}}
				},
			},
			{{end}}
		})
	{{end}}
	rev.DefaultValidationKeys = map[string]map[int]string{ {{range $path, $lines := .ValidationKeys}}
		"{{$path}}": { {{range $line, $key := $lines}}
			{{$line}}: "{{$key}}",{{end}}
		},{{end}}
	}
	rev.UnitTests = []interface{}{ {{range .UnitTests}}
		(*{{.PackageName}}.{{.StructName}})(nil),{{end}}
	}
	rev.FunctionalTests = []interface{}{ {{range .FunctionalTests}}
		(*{{.PackageName}}.{{.StructName}})(nil),{{end}}
	}

	rev.Run(*port)
}
`
