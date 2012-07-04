// The Harness for a Revel program.
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
	"github.com/howeyc/fsnotify"
	"github.com/robfig/revel"
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
	"strings"
	"text/template"
)

const REGISTER_CONTROLLERS = `
// target: {{.AppName}}
package main

import (
	"flag"
	"reflect"
	"github.com/robfig/revel"
	{{range .ImportPaths}}
  "{{.}}"
  {{end}}
)

var (
	port *int = flag.Int("port", 0, "Port")
	importPath *string = flag.String("importPath", "", "Path to the app.")

	// So compiler won't complain if the generated code doesn't reference reflect package...
	_ = reflect.Invalid
)

func main() {
	rev.LOG.Println("Running revel server")
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
	rev.Run(*port)
}
`

// Reverse proxy requests to the application server.
// On each request, proxy sends (NotifyRequest = true)
// If code change has been detected in app:
// - app is rebuilt and restarted, send proxy (NotifyReady = true)
// - else, send proxy (NotifyReady = true)

type harnessProxy struct {
	serverHost    string
	proxy         *httputil.ReverseProxy
	NotifyRequest chan bool  // Strobed on every request.
	NotifyReady   chan error // Strobed when request may proceed.
}

func renderError(w http.ResponseWriter, r *http.Request, err error) {
	rev.RenderError(rev.NewRequest(r), rev.NewResponse(w), err)
}

func (hp *harnessProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// First, poll to see if there's a pending error in NotifyReady
	select {
	case err := <-hp.NotifyReady:
		if err != nil {
			renderError(w, r, err)
		}
	default:
		// Usually do nothing.
	}

	// Notify that a request is coming through, and wait for the go-ahead.
	hp.NotifyRequest <- true
	err := <-hp.NotifyReady

	// If an error was returned, create the page and show it to the user.
	if err != nil {
		renderError(w, r, err)
		return
	}

	// Reverse proxy the request.
	// (Need special code for websockets, courtesy of bradfitz)
	if r.Header.Get("Upgrade") == "websocket" {
		proxyWebsocket(w, r, hp.serverHost)
	} else {
		hp.proxy.ServeHTTP(w, r)
	}
}

// ReverseProxy doesn't work with websocket requests.
// This function copies data between websocket client and server until one side
// closes the connection.
func proxyWebsocket(w http.ResponseWriter, r *http.Request, host string) {
	d, err := net.Dial("tcp", host)
	if err != nil {
		http.Error(w, "Error contacting backend server.", 500)
		log.Printf("Error dialing websocket backend %s: %v", host, err)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Not a hijacker?", 500)
		return
	}
	nc, _, err := hj.Hijack()
	if err != nil {
		log.Printf("Hijack error: %v", err)
		return
	}
	defer nc.Close()
	defer d.Close()

	err = r.Write(d)
	if err != nil {
		log.Printf("Error copying request to target: %v", err)
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

func startReverseProxy(port int) *harnessProxy {
	serverUrl, _ := url.ParseRequestURI(fmt.Sprintf("http://localhost:%d", port))
	reverseProxy := &harnessProxy{
		serverHost:    serverUrl.String()[len("http://"):],
		proxy:         httputil.NewSingleHostReverseProxy(serverUrl),
		NotifyRequest: make(chan bool),
		NotifyReady:   make(chan error),
	}
	go func() {
		appPort := getAppPort()
		log.Println("Listening on port", appPort)
		err := http.ListenAndServe(fmt.Sprintf(":%d", appPort), reverseProxy)
		if err != nil {
			log.Fatalln("Failed to start reverse proxy:", err)
		}
	}()
	return reverseProxy
}

func getAppPort() int {
	port, err := rev.Config.Int("http.port")
	if err != nil {
		log.Println("Parsing http.port failed:", err)
		return 9000
	}
	return port
}

var (
	// Will not watch directories with these names (or their subdirectories)
	DoNotWatch = []string{"tmp", "views"}
)

func Run(mode string) {

	// If we are in prod mode, just build and run the application.
	if mode == rev.PROD {
		log.Println("Building...")
		if err := rebuild(getAppPort()); err != nil {
			log.Fatalln(err)
		}
		cmd.Wait()
		return
	}

	// Get a template loader to render errors.
	// Prefer the app's views/errors directory, and fall back to the stock error pages.
	rev.MainTemplateLoader = rev.NewTemplateLoader(
		rev.ViewsPath,
		rev.RevelTemplatePath)

	// Get a port on which to run the application
	port := getFreePort()

	// Run a reverse proxy to it.
	proxy := startReverseProxy(port)

	// Listen for changes to the user app.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	watcher.Event = make(chan *fsnotify.FileEvent, 10)
	watcher.Error = make(chan error, 10)

	// Listen to all app subdirectories (except /views)
	filepath.Walk(rev.AppPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			rev.LOG.Println("error walking app:", err)
			return nil
		}
		if info.IsDir() {
			if rev.ContainsString(DoNotWatch, info.Name()) {
				return filepath.SkipDir
			}
			err = watcher.Watch(path)
			rev.LOG.Println("Watching:", path)
			if err != nil {
				rev.LOG.Println("Failed to watch", path, ":", err)
			}
		}
		return nil
	})

	// Define an exit handler that kills the revel server (since it won't die on
	// its own, if the harness exits)
	defer func() {
		if cmd != nil {
			cmd.Process.Kill()
			cmd = nil
		}
	}()

	// Start the listen / rebuild loop.
	var dirty bool = true
	for {
		err = nil

		// It spins in this loop for each inotify change, and each request.
		// If there is a request after an inotify change, it breaks out to rebuild.
		for {
			select {
			case ev := <-watcher.Event:
				// Ignore changes to dot-files.
				if !strings.HasPrefix(path.Base(ev.Name), ".") {
					log.Println(ev)
					dirty = true
				}
				continue
			case err = <-watcher.Error:
				log.Println("Inotify error:", err)
				continue
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

// Rebuild the Revel application and run it on the given port.
func rebuild(port int) (compileError *rev.Error) {
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
		log.Println("Killing revel server pid", cmd.Process.Pid)
		err := cmd.Process.Kill()
		if err != nil {
			log.Fatalln("Failed to kill revel server:", err)
		}
	}

	// Create a fresh temp dir.
	tmpPath := path.Join(rev.AppPath, "tmp")
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
	// It relies on the user having "go" installed.
	goPath, err := exec.LookPath("go")
	if err != nil {
		log.Fatalf("Go executable not found in PATH.")
	}

	ctx := build.Default
	pkg, err := ctx.Import(rev.ImportPath, "", build.FindOnly)
	if err != nil {
		log.Fatalf("Failure importing", rev.ImportPath)
	}
	binName := path.Join(pkg.BinDir, rev.AppName)
	buildCmd := exec.Command(goPath, "build", "-o", binName, path.Join(rev.ImportPath, "app", "tmp"))
	output, err := buildCmd.CombinedOutput()

	// If we failed to build, parse the error message.
	if err != nil {
		return newCompileError(output)
	}

	// Run the server, via tmp/main.go.
	cmd = exec.Command(binName,
		fmt.Sprintf("-port=%d", port),
		fmt.Sprintf("-importPath=%s", rev.ImportPath))
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
		log.Println("Failed to parse build errors:\n", string(output))
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
		log.Println(compileError.MetaError)
		return compileError
	}

	compileError.SourceLines = fileStr
	return compileError
}
