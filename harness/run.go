package harness

import (
	"bytes"
	"fmt"
	"github.com/robfig/revel"
	"go/build"
	"io"
	"net"
	"net/http"
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
)

// Run the Revel program, optionally using the harness.
func Run(useHarness bool) {
	// If we are in prod mode, just build and run the application.
	if ! useHarness {
		rev.INFO.Println("Building...")
		if err := rebuild(getAppAddress(), getAppPort()); err != nil {
			rev.ERROR.Fatalln(err)
		}
		cmd.Wait()
		return
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
}

// Rebuild the Revel application and run it on the given port.
func rebuild(addr string, port int) (compileError *rev.Error) {
	rev.TRACE.Println("Rebuild")
	controllerSpecs, compileError := ScanControllers(path.Join(rev.AppPath, "controllers"))
	if compileError != nil {
		return compileError
	}

	tmpl := template.New("RegisterControllers")
	tmpl = template.Must(tmpl.Parse(REGISTER_CONTROLLERS))
	var registerControllerSource string = rev.ExecuteTemplate(tmpl, map[string]interface{}{
		"AppName":     rev.AppName,
		"Controllers": controllerSpecs,
		"ImportPaths": uniqueImportPaths(controllerSpecs),
		"RunMode":     rev.RunMode,
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
		rev.ERROR.Fatalf("Failure importing", rev.ImportPath)
	}
	binName := path.Join(pkg.BinDir, rev.AppName)
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	buildCmd := exec.Command(goPath, "build", "-o", binName, path.Join(rev.ImportPath, "app", "tmp"))
	rev.TRACE.Println("Exec build:", buildCmd.Path, buildCmd.Args)
	output, err := buildCmd.CombinedOutput()

	// If we failed to build, parse the error message.
	if err != nil {
		return newCompileError(output)
	}

	// Run the server, via tmp/main.go.
	cmd = exec.Command(binName,
		fmt.Sprintf("-addr=%s", addr),
		fmt.Sprintf("-port=%d", port),
		fmt.Sprintf("-importPath=%s", rev.ImportPath))
	rev.TRACE.Println("Exec app:", cmd.Path, cmd.Args)
	listeningWriter := startupListeningWriter{os.Stdout, make(chan bool)}
	cmd.Stdout = listeningWriter
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		rev.ERROR.Fatalln("Error running:", err)
	}

	<-listeningWriter.notifyReady
	return nil
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
func uniqueImportPaths(specs []*ControllerSpec) (paths []string) {
	importPathMap := make(map[string]bool)
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

// proxyWebsocket copies data between websocket client and server until one side
// closes the connection.  (ReverseProxy doesn't work with websocket requests.)
func proxyWebsocket(w http.ResponseWriter, r *http.Request, host string) {
	d, err := net.Dial("tcp", host)
	if err != nil {
		http.Error(w, "Error contacting backend server.", 500)
		rev.ERROR.Printf("Error dialing websocket backend %s: %v", host, err)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Not a hijacker?", 500)
		return
	}
	nc, _, err := hj.Hijack()
	if err != nil {
		rev.ERROR.Printf("Hijack error: %v", err)
		return
	}
	defer nc.Close()
	defer d.Close()

	err = r.Write(d)
	if err != nil {
		rev.ERROR.Printf("Error copying request to target: %v", err)
		return
	}

	errc := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errc <- err
	}
	go cp(d, nc)
	go cp(nc, d)
	<-errc
}

const REGISTER_CONTROLLERS = `package main

import (
	"flag"
	"reflect"
	"github.com/robfig/revel"
	{{range .ImportPaths}}
  "{{.}}"
  {{end}}
)

var (
	addr *string = flag.String("addr", "", "Address to listen on")
	port *int = flag.Int("port", 0, "Port")
	importPath *string = flag.String("importPath", "", "Path to the app.")

	// So compiler won't complain if the generated code doesn't reference reflect package...
	_ = reflect.Invalid
)

func main() {
	rev.INFO.Println("Running revel server")
	flag.Parse()
	rev.Init(*importPath, "{{.RunMode}}")
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
	rev.Run(*addr, *port)
}
`
