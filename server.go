package revel

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	MainRouter         *Router
	MainTemplateLoader *TemplateLoader
	MainWatcher        *Watcher
	Server             *http.Server
)

// This method handles all requests.  It dispatches to handleInternal after
// handling / adapting websocket connections.
func handle(w http.ResponseWriter, r *http.Request) {
	upgrade := r.Header.Get("Upgrade")
	if upgrade == "websocket" || upgrade == "Websocket" {
		websocket.Handler(func(ws *websocket.Conn) {
			r.Method = "WS"
			handleInternal(w, r, ws)
		}).ServeHTTP(w, r)
	} else {
		handleInternal(w, r, nil)
	}
}

func handleInternal(w http.ResponseWriter, r *http.Request, ws *websocket.Conn) {
	var (
		req  = NewRequest(r)
		resp = NewResponse(w)
		c    = NewController(req, resp)
	)
	req.Websocket = ws

	Filters[0](c, Filters[1:])
	if c.Result != nil {
		c.Result.Apply(req, resp)
	}
}

// Run the server.
// This is called from the generated main file.
// If port is non-zero, use that.  Else, read the port from app.conf.
func Run(port int) {
	address := HttpAddr
	if port == 0 {
		port = HttpPort
	}

	var network = "tcp"
	var localAddress string

	// If the port is zero, treat the address as a fully qualified local address.
	// This address must be prefixed with the network type followed by a colon,
	// e.g. unix:/tmp/app.socket or tcp6:::1 (equivalent to tcp6:0:0:0:0:0:0:0:1)
	if port == 0 {
		parts := strings.SplitN(address, ":", 2)
		network = parts[0]
		localAddress = parts[1]
	} else {
		localAddress = address + ":" + strconv.Itoa(port)
	}

	MainTemplateLoader = NewTemplateLoader(TemplatePaths)

	// The "watch" config variable can turn on and off all watching.
	// (As a convenient way to control it all together.)
	if Config.BoolDefault("watch", true) {
		MainWatcher = NewWatcher()
		Filters = append([]Filter{WatchFilter}, Filters...)
	}

	// If desired (or by default), create a watcher for templates and routes.
	// The watcher calls Refresh() on things on the first request.
	if MainWatcher != nil && Config.BoolDefault("watch.templates", true) {
		MainWatcher.Listen(MainTemplateLoader, MainTemplateLoader.paths...)
	} else {
		MainTemplateLoader.Refresh()
	}

	Server = &http.Server{
		Addr:    localAddress,
		Handler: http.HandlerFunc(handle),
	}

	runStartupHooks()

	go func() {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Listening on %s...\n", localAddress)
	}()

	if HttpSsl {
		if network != "tcp" {
			// This limitation is just to reduce complexity, since it is standard
			// to terminate SSL upstream when using unix domain sockets.
			ERROR.Fatalln("SSL is only supported for TCP sockets. Specify a port to listen on.")
		}
		ERROR.Fatalln("Failed to listen:",
			Server.ListenAndServeTLS(HttpSslCert, HttpSslKey))
	} else {
		listener, err := net.Listen(network, localAddress)
		if err != nil {
			ERROR.Fatalln("Failed to listen:", err)
		}
		ERROR.Fatalln("Failed to serve:", Server.Serve(listener))
	}
}

func runStartupHooks() {
	for _, hook := range startupHooks {
		hook()
	}
}

var startupHooks []func()

// Register a function to be run at app startup.
//
// The order you register the functions will be the order they are run.
// You can think of it as a FIFO queue.
// This process will happen after the config file is read
// and before the server is listening for connections.
//
// Ideally, your application should have only one call to init() in the file init.go.
// The reason being that the call order of multiple init() functions in
// the same package is undefined.
// Inside of init() call revel.OnAppStart() for each function you wish to register.
//
// Example:
//
//      // from: yourapp/app/controllers/somefile.go
//      func InitDB() {
//          // do DB connection stuff here
//      }
//
//      func FillCache() {
//          // fill a cache from DB
//          // this depends on InitDB having been run
//      }
//
//      // from: yourapp/app/init.go
//      func init() {
//          // set up filters...
//
//          // register startup functions
//          revel.OnAppStart(InitDB)
//          revel.OnAppStart(FillCache)
//      }
//
// This can be useful when you need to establish connections to databases or third-party services,
// setup app components, compile assets, or any thing you need to do between starting Revel and accepting connections.
//
func OnAppStart(f func()) {
	startupHooks = append(startupHooks, f)
}
