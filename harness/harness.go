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
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"play"
)

const REGISTER_CONTROLLERS = `
// target: {{.AppName}}
package main

import (
	"flag"
	"play"
	{{range $k, $v := .controllers}}
  "{{$k}}"
  {{end}}
)

var port *int = flag.Int("port", 0, "Port")

func main() {
	play.LOG.Println("Running play server")
	flag.Parse()
  {{range $k, $v := .controllers}}
  {{range $v}}
	play.RegisterController((*{{.PackageName}}.{{.StructName}})(nil))
  {{end}}
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
	NotifyReady chan bool  // Strobed when request may proceed.
}

func (hp *harnessProxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	hp.NotifyRequest <- true
	<- hp.NotifyReady
	hp.proxy.ServeHTTP(wr, req)
}

func startReverseProxy(port int) *harnessProxy {
	serverUrl, _ := url.ParseRequest(fmt.Sprintf("http://localhost:%d", port))
	reverseProxy := &harnessProxy{
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
		NotifyRequest: make(chan bool),
		NotifyReady: make(chan bool),
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

	// Build the application, and run it on that port.
	rebuild(port)

	// Start the listen / rebuild loop.
	for {

		// It spins in this loop for each inotify change, and each request.
		// If there is a request after an inotify change, it breaks out to rebuild.
		dirty := false
		for {
			select {
			case ev := <-watcher.Event:
				log.Println("Detected change to application directories:", ev.DirNames)
				dirty = true
				continue
			case err := <-watcher.Error:
				log.Fatalf("Inotify error: %s", err)
			case _ = <-proxy.NotifyRequest:
				if !dirty {
					proxy.NotifyReady <- true
					continue
				}
			}

			break
		}

		// There has been a change to the app and a new request is pending.
		// Rebuild it and send the "ready" signal.
		log.Println("Rebuild")
		rebuild(port)
		dirty = false
		proxy.NotifyReady <- true
	}
}

var cmd *exec.Cmd

// Rebuild the Play! application and run it on the given port.
func rebuild(port int) {
	tmpl := template.New("RegisterControllers")
	tmpl = template.Must(tmpl.Parse(REGISTER_CONTROLLERS))
	var registerControllerSource string = play.ExecuteTemplate(tmpl, map[string]interface{} {
		"AppName": play.AppName,
		"controllers": listControllers(filepath.Join(play.AppPath, "controllers")),
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
	tmpPath := filepath.Join(play.AppPath, "tmp")
	err := os.RemoveAll(tmpPath)
	if err != nil {
		log.Println("Failed to remove tmp dir:", err)
	}
	err = os.Mkdir(tmpPath, 0777)
	if err != nil {
		log.Fatalf("Failed to make tmp directory: %v", err)
	}

	// Create the new file
	controllersFile, err := os.Create(filepath.Join(tmpPath, "main.go"))
	if err != nil {
		log.Fatalf("Failed to create main.go: %v", err)
	}
	_, err = controllersFile.WriteString(registerControllerSource)
	if err != nil {
		log.Fatalf("Failed to write to main.go: %v", err)
	}

	// Build the user program (all code under app).
	// It relies on the user having gb installed.
	gbPath, err := exec.LookPath("gb")
	if err != nil {
		log.Fatalf("GB executable not found in PATH.  Please goinstall it.")
	}
	cmd := exec.Command(gbPath, path.Join(play.AppPath, "tmp"))
	err = cmd.Run()
	if err != nil {
		output, _ := cmd.CombinedOutput()
		log.Fatalln("Failed to build app:\n%s", string(output))
	}

	// Run the server, via tmp/main.go.
	appTree, _, _ := build.FindTree(play.AppPath)
	cmd = exec.Command(filepath.Join(appTree.BinDir(), "tmp"), fmt.Sprintf("-port=%d", port))
	listeningWriter := startupListeningWriter{os.Stdout, make(chan bool)}
	cmd.Stdout = listeningWriter
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Fatalln("Error running:", err)
	}

	<-listeningWriter.notifyReady
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

type typeName struct {
	PackageName, StructName string
}

func listControllers(path string) (controllerTypeNames map[string][]*typeName) {
	controllerTypeNames = make(map[string][]*typeName)

	// Parse files within the path.
	var pkgs map[string]*ast.Package
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, func(f os.FileInfo) bool { return !f.IsDir() }, 0)
	if err != nil {
		log.Fatalf("Failed to parse dir: %s", err)
	}

	// For each package... (often only "controllers")
	for _, pkg := range pkgs {

		// For each source file in the package...
		for _, file := range pkg.Files {

			// For each declaration in the source file...
			for _, decl := range file.Decls {
				// Find Type declarations that embed *play.Controller.
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				if genDecl.Tok != token.TYPE {
					continue
				}

				if len(genDecl.Specs) != 1 {
					play.LOG.Printf("Surprising: Decl does not have 1 Spec: %v", genDecl)
					continue
				}

				spec := genDecl.Specs[0].(*ast.TypeSpec)
				structType, ok := spec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				var fieldList []*ast.Field = structType.Fields.List
				if len(fieldList) == 0 {
					continue
				}

				// Look for ast.Field to have type StarExpr { SelectorExpr { "play", "Controller" } }
				starExpr, ok := fieldList[0].Type.(*ast.StarExpr)
				if !ok {
					continue
				}

				selectorExpr, ok := starExpr.X.(*ast.SelectorExpr)
				if !ok {
					continue
				}

				pkgIdent, ok := selectorExpr.X.(*ast.Ident)
				if !ok {
					continue
				}

				// TODO: Support sub-types of play.Controller as well.
				if pkgIdent.Name == "play" && selectorExpr.Sel.Name == "Controller" {
					fullImportPath := play.BaseImportPath + "/" + play.AppName + "/app/" + pkg.Name
					controllerTypeNames[fullImportPath] = append(controllerTypeNames[fullImportPath],
						&typeName{ pkg.Name, spec.Name.Name })
				}
			}
		}
	}
	return
}
