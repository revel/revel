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
	"fmt"
	"github.com/robfig/revel"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
)

var (
	watcher    *rev.Watcher
	doNotWatch = []string{"tmp", "views"}

	lastRequestHadError int32
)

// Harness reverse proxies requests to the application server.
// It builds / runs / rebuilds / restarts the server when code is changed.
type Harness struct {
	app        *App
	serverHost string
	port       int
	proxy      *httputil.ReverseProxy
}

func renderError(w http.ResponseWriter, r *http.Request, err error) {
	rev.RenderError(rev.NewRequest(r), rev.NewResponse(w), err)
}

// ServeHTTP handles all requests.
// It checks for changes to app, rebuilds if necessary, and forwards the request.
func (hp *Harness) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Don't rebuild the app for favicon requests.
	if lastRequestHadError > 0 && r.URL.Path == "/favicon.ico" {
		return
	}

	// Flush any change events and rebuild app if necessary.
	// Render an error page if the rebuild / restart failed.
	err := watcher.Notify()
	if err != nil {
		atomic.CompareAndSwapInt32(&lastRequestHadError, 0, 1)
		renderError(w, r, err)
		return
	}
	atomic.CompareAndSwapInt32(&lastRequestHadError, 1, 0)

	// Reverse proxy the request.
	// (Need special code for websockets, courtesy of bradfitz)
	if r.Header.Get("Upgrade") == "websocket" {
		proxyWebsocket(w, r, hp.serverHost)
	} else {
		hp.proxy.ServeHTTP(w, r)
	}
}

// Return a reverse proxy that forwards requests to the given port.
func NewHarness() *Harness {
	// Get a template loader to render errors.
	// Prefer the app's views/errors directory, and fall back to the stock error pages.
	rev.MainTemplateLoader = rev.NewTemplateLoader(rev.TemplatePaths)
	rev.MainTemplateLoader.Refresh()

	addr := rev.HttpAddr
	port := rev.Config.IntDefault("harness.port", 0)

	if port == 0 {
		port = getFreePort(addr)
	}

	serverUrl, _ := url.ParseRequestURI(fmt.Sprintf("http://%s:%d", addr, port))

	harness := &Harness{
		port:       port,
		serverHost: serverUrl.String()[len("http://"):],
		proxy:      httputil.NewSingleHostReverseProxy(serverUrl),
	}
	return harness
}

// Rebuild the Revel application and run it on the given port.
func (h *Harness) Refresh() (err *rev.Error) {
	if h.app != nil {
		h.app.Kill()
	}

	rev.TRACE.Println("Rebuild")
	h.app, err = Build()
	if err != nil {
		return
	}

	h.app.Port = h.port
	h.app.Cmd().Start()
	return
}

func (h *Harness) WatchDir(info os.FileInfo) bool {
	return !rev.ContainsString(doNotWatch, info.Name())
}

func (h *Harness) WatchFile(filename string) bool {
	return strings.HasSuffix(filename, ".go")
}

// Run the harness, which listens for requests and proxies them to the app
// server, which it runs and rebuilds as necessary.
func (h *Harness) Run() {
	watcher = rev.NewWatcher()
	watcher.Listen(h, rev.CodePaths...)

	rev.INFO.Printf("Listening on %s:%d", rev.HttpAddr, rev.HttpPort)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", rev.HttpAddr, rev.HttpPort), h)
	if err != nil {
		rev.ERROR.Fatalln("Failed to start reverse proxy:", err)
	}
}

// Find an unused port
func getFreePort(addr string) (port int) {
	conn, err := net.Listen("tcp", fmt.Sprintf("%s:0", addr))
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
