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
	"crypto/tls"
	"fmt"
	"github.com/revel/revel"
	"go/build"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
)

var (
	watcher    *revel.Watcher
	doNotWatch = []string{"tmp", "views", "routes"}

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
	req, resp := revel.NewRequest(r), revel.NewResponse(w)
	c := revel.NewController(req, resp)
	c.RenderError(err).Apply(req, resp)
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
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		proxyWebsocket(w, r, hp.serverHost)
	} else {
		hp.proxy.ServeHTTP(w, r)
	}
}

// Return a reverse proxy that forwards requests to the given port.
func NewHarness() *Harness {
	// Get a template loader to render errors.
	// Prefer the app's views/errors directory, and fall back to the stock error pages.
	revel.MainTemplateLoader = revel.NewTemplateLoader(
		[]string{path.Join(revel.RevelPath, "templates")})
	revel.MainTemplateLoader.Refresh()

	addr := revel.HttpAddr
	port := revel.Config.IntDefault("harness.port", 0)
	scheme := "http"
	if revel.HttpSsl {
		scheme = "https"
	}

	// If the server is running on the wildcard address, use "localhost"
	if addr == "" {
		addr = "localhost"
	}

	if port == 0 {
		port = getFreePort()
	}

	serverUrl, _ := url.ParseRequestURI(fmt.Sprintf(scheme+"://%s:%d", addr, port))

	harness := &Harness{
		port:       port,
		serverHost: serverUrl.String()[len(scheme+"://"):],
		proxy:      httputil.NewSingleHostReverseProxy(serverUrl),
	}

	if revel.HttpSsl {
		harness.proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return harness
}

// Rebuild the Revel application and run it on the given port.
func (h *Harness) Refresh() (err *revel.Error) {
	if h.app != nil {
		h.app.Kill()
	}

	revel.TRACE.Println("Rebuild")
	h.app, err = Build()
	if err != nil {
		return
	}

	h.app.Port = h.port
	if err2 := h.app.Cmd().Start(); err2 != nil {
		return &revel.Error{
			Title:       "App failed to start up",
			Description: err2.Error(),
		}
	}

	return
}

func (h *Harness) WatchDir(info os.FileInfo) bool {
	return !revel.ContainsString(doNotWatch, info.Name())
}

func (h *Harness) WatchFile(filename string) bool {
	return strings.HasSuffix(filename, ".go")
}

// Run the harness, which listens for requests and proxies them to the app
// server, which it runs and rebuilds as necessary.
func (h *Harness) Run() {
	var paths []string
	if revel.Config.BoolDefault("watch.gopath", false) {
		gopaths := filepath.SplitList(build.Default.GOPATH)
		paths = append(paths, gopaths...)
	}
	paths = append(paths, revel.CodePaths...)
	watcher = revel.NewWatcher()
	watcher.Listen(h, paths...)

	go func() {
		addr := fmt.Sprintf("%s:%d", revel.HttpAddr, revel.HttpPort)
		revel.INFO.Printf("Listening on %s", addr)

		var err error
		if revel.HttpSsl {
			err = http.ListenAndServeTLS(addr, revel.HttpSslCert,
				revel.HttpSslKey, h)
		} else {
			err = http.ListenAndServe(addr, h)
		}
		if err != nil {
			revel.ERROR.Fatalln("Failed to start reverse proxy:", err)
		}
	}()

	// Kill the app on signal.
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch
	if h.app != nil {
		h.app.Kill()
	}
	os.Exit(1)
}

// Find an unused port
func getFreePort() (port int) {
	conn, err := net.Listen("tcp", ":0")
	if err != nil {
		revel.ERROR.Fatal(err)
	}

	port = conn.Addr().(*net.TCPAddr).Port
	err = conn.Close()
	if err != nil {
		revel.ERROR.Fatal(err)
	}
	return port
}

// proxyWebsocket copies data between websocket client and server until one side
// closes the connection.  (ReverseProxy doesn't work with websocket requests.)
func proxyWebsocket(w http.ResponseWriter, r *http.Request, host string) {
	d, err := net.Dial("tcp", host)
	if err != nil {
		http.Error(w, "Error contacting backend server.", 500)
		revel.ERROR.Printf("Error dialing websocket backend %s: %v", host, err)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Not a hijacker?", 500)
		return
	}
	nc, _, err := hj.Hijack()
	if err != nil {
		revel.ERROR.Printf("Hijack error: %v", err)
		return
	}
	defer nc.Close()
	defer d.Close()

	err = r.Write(d)
	if err != nil {
		revel.ERROR.Printf("Error copying request to target: %v", err)
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
