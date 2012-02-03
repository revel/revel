// The Harness for a GoPlay! program.
//
// It has a couple responsibilities:
// 1. Parse the user program, generating a main.go file that registers
//    controller classes and starts the user's server.
// 2. Build and run the user program.  Show compile errors.
// 3. Monitor the user source and re-build / restart the program when necessary.
//
// Source files are generated in the app/tmp directory.

package harness

import (
	"bytes"
	"fmt"
	"go/build"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"text/template"
	"play"
)

const REGISTER_CONTROLLERS = `
// target: {{.AppName}}
package main

import (
	"flag"
	"reflect"
	"play"
	{{range .ImportPaths}}
  "{{.}}"
  {{end}}
)

var (
	port *int = flag.Int("port", 0, "Port")
	importPath *string = flag.String("importPath", "", "Path to the app.")
)

func main() {
	play.LOG.Println("Running play server")
	flag.Parse()
	play.Init(*importPath)
	{{range $i, $c := .Controllers}}
	play.RegisterController((*{{.PackageName}}.{{.StructName}})(nil),
		[]*play.MethodType{
			{{range .MethodSpecs}}&play.MethodType{
				Name: "{{.Name}}",
				Args: []*play.MethodArg{ {{range .Args}}
					&play.MethodArg{Name: "{{.Name}}", Type: reflect.TypeOf((*{{.TypeName}})(nil)) },{{end}}
			  },
			},
			{{end}}
		})
	{{end}}
	play.Run(*port)
}
`

// Reverse proxy requests to the application server.
// On each request, proxy sends (NotifyRequest = true)
// If code change has been detected in app:
// - app is rebuilt and restarted, send proxy (NotifyReady = true)
// - else, send proxy (NotifyReady = true)

type harnessProxy struct {
	proxy *httputil.ReverseProxy
	NotifyRequest chan bool  // Strobed on every request.
	NotifyReady chan error  // Strobed when request may proceed.
}

func (hp *harnessProxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {

	// First, poll to see if there's a pending error in NotifyReady
	select {
	case err := <-hp.NotifyReady:
		serveError(wr, req, err)
	default:
		// Usually do nothing.
	}

	// Notify that a request is coming through, and wait for the go-ahead.
	hp.NotifyRequest <- true
	err := <- hp.NotifyReady

	// If an error was returned, create the page and show it to the user.
	if err != nil {
		serveError(wr, req, err)
		return
	}

	// Reverse proxy the request.
	hp.proxy.ServeHTTP(wr, req)
}

func serveError(wr http.ResponseWriter, req *http.Request, err error) {
	switch e := err.(type) {
	case *play.CompileError:
		play.RenderError(wr, e)
	default:
		play.RenderError(wr, map[string]string {
			"Title": "Unexpected error",
			"Path": "(unknown)",
			"Description": "An unexpected error occurred: " + err.Error(),
		})
	}
}

func startReverseProxy(port int) *harnessProxy {
	serverUrl, _ := url.ParseRequest(fmt.Sprintf("http://localhost:%d", port))
	reverseProxy := &harnessProxy{
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
		NotifyRequest: make(chan bool),
		NotifyReady: make(chan error),
	}
	go func() {
		err := http.ListenAndServe(":9000", reverseProxy)
		if err != nil {
			log.Fatalln("Failed to start reverse proxy:", err)
		}
	}()
	return reverseProxy
}

func Run() {

	// Get a port on which to run the application
	port := getFreePort()

	// Run a reverse proxy to it.
	proxy := startReverseProxy(port)

	// Listen for changes to the user app.
	watcher := NewWatcher(play.AppPath)

	// Define an exit handler that kills the play server (since it won't die on
	// its own, if the harness exits)
	defer func() {
		if cmd != nil {
			cmd.Process.Kill()
			cmd = nil
		}
	}()

	// Start the listen / rebuild loop.
	var err error = nil
	var dirty bool = true
	for {
		err = nil

		// It spins in this loop for each inotify change, and each request.
		// If there is a request after an inotify change, it breaks out to rebuild.
		for {
			select {
			case ev := <-watcher.Event:
				log.Println("Detected change to application directories:", ev.DirNames)
				dirty = true
				continue
			case err = <-watcher.Error:
				log.Fatalf("Inotify error: %s", err)
			case _ = <-proxy.NotifyRequest:
				if !dirty {
					proxy.NotifyReady <- nil
					continue
				}
			}

			break
		}

		// There has been a change to the app and a new request is pending.
		// Rebuild it and send the "ready" signal.
		log.Println("Rebuild")
		err := rebuild(port)
		if err != nil {
			log.Println(err.Error())
			proxy.NotifyReady <- err
			continue
		}
		dirty = false
		proxy.NotifyReady <- nil
	}
}

var cmd *exec.Cmd

// Rebuild the Play! application and run it on the given port.
func rebuild(port int) (compileError *play.CompileError) {
	controllerSpecs, compileError := ScanControllers(path.Join(play.AppPath, "controllers"))
	if compileError != nil {
		return compileError
	}

	tmpl := template.New("RegisterControllers")
	tmpl = template.Must(tmpl.Parse(REGISTER_CONTROLLERS))
	var registerControllerSource string = play.ExecuteTemplate(tmpl, map[string]interface{} {
		"AppName": play.AppName,
		"Controllers": controllerSpecs,
		"ImportPaths": uniqueImportPaths(controllerSpecs),
	})

	// Terminate the server if it's already running.
	if cmd != nil {
		log.Println("Killing play server pid", cmd.Process.Pid)
		err := cmd.Process.Kill()
		if err != nil {
			log.Fatalln("Failed to kill play server:", err)
		}
	}

	// Create a fresh temp dir.
	tmpPath := path.Join(play.AppPath, "tmp")
	err := os.RemoveAll(tmpPath)
	if err != nil {
		log.Println("Failed to remove tmp dir:", err)
	}
	err = os.Mkdir(tmpPath, 0777)
	if err != nil {
		log.Fatalf("Failed to make tmp directory: %v", err)
	}

	// Create the new file
	controllersFile, err := os.Create(path.Join(tmpPath, "main.go"))
	if err != nil {
		log.Fatalf("Failed to create main.go: %v", err)
	}
	_, err = controllersFile.WriteString(registerControllerSource)
	if err != nil {
		log.Fatalf("Failed to write to main.go: %v", err)
	}

	// Build the user program (all code under app).
	// It relies on the user having gb installed.
	goPath, err := exec.LookPath("go")
	if err != nil {
		log.Fatalf("GB executable not found in PATH.  Please goinstall it.")
	}

	appTree, _, _ := build.FindTree(play.AppPath)
	binName := path.Join(appTree.BinDir(), play.AppName)
	cmd := exec.Command(goPath, "build", "-o", binName, path.Join(play.AppPath, "tmp"))
	output, err := cmd.Output()

	// If we failed to build, parse the error message.
	if err != nil {
		return newCompileError(output)
	}

	// Run the server, via tmp/main.go.
	cmd = exec.Command(binName,
		fmt.Sprintf("-port=%d", port),
		fmt.Sprintf("-importPath=%s", play.ImportPath))
	listeningWriter := startupListeningWriter{os.Stdout, make(chan bool)}
	cmd.Stdout = listeningWriter
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Fatalln("Error running:", err)
	}

	<-listeningWriter.notifyReady
	return nil
}

// A io.Writer that copies to the destination, and listens for "Listening on.."
// in the stream.  (Which tells us when the play server has finished starting up)
// This is super ghetto, but by far the simplest thing that should work.
type startupListeningWriter struct {
	dest io.Writer
	notifyReady chan bool
}

func (w startupListeningWriter) Write(p []byte) (n int, err error) {
	if w.notifyReady != nil && bytes.Contains(p, []byte("Listening")) {
		w.notifyReady <- true
		w.notifyReady = nil
	}
	return w.dest.Write(p)
}

// Find an unused port
func getFreePort() (port int) {
	conn, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	port = conn.Addr().(*net.TCPAddr).Port
	err = conn.Close()
	if err != nil {
		log.Fatal(err)
	}
	return port
}

func uniqueImportPaths(specs []*ControllerSpec) (paths []string) {
	importPathMap := make(map[string]bool)
	for _, spec := range specs {
		importPathMap[spec.ImportPath] = true
	}
	for importPath, _ := range importPathMap {
		paths = append(paths, importPath)
	}
	return
}

// Parse the output of the "gb" compile command.
// Return a detailed CompileError.
func newCompileError(output []byte) *play.CompileError {
	errorMatch := regexp.MustCompile(`(?m)^(.+):(\d+): (.*)$`).
		FindSubmatch(output)
	if errorMatch == nil {
		log.Println("Failed to parse build errors:\n", string(output))
		return &play.CompileError{
			SourceType: "Go code",
			Title: "Go Compilation Error",
			Description: "See console for build error.",
		}
	}

	// Read the source for the offending file.
	var (
		relFilename = string(errorMatch[1])  // e.g. "src/play/sample/app/controllers/app.go"
		absFilename, _ = filepath.Abs(relFilename)
		line, _ = strconv.Atoi(string(errorMatch[2]))
		description = string(errorMatch[3])
		compileError = &play.CompileError{
			SourceType: "Go code",
			Title: "Go Compilation Error",
			Path: relFilename,
			Description: description,
			Line: line,
		}
	)

	fileStr, err := play.ReadLines(absFilename)
	if err != nil {
		compileError.MetaError = absFilename + ": " + err.Error()
		log.Println(compileError.MetaError)
		return compileError
	}

	compileError.SourceLines = fileStr
	return compileError
}
