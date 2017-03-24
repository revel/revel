// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

// Revel's variables server, router, etc
var (
	MainRouter         *Router
	MainTemplateLoader *TemplateLoader
	MainWatcher        *Watcher
	Server             *http.Server
)

// This method handles all requests.  It dispatches to handleInternal after
// handling / adapting websocket connections.
func handle(w http.ResponseWriter, r *http.Request) {
	if maxRequestSize := int64(Config.IntDefault("http.maxrequestsize", 0)); maxRequestSize > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	}

	upgrade := r.Header.Get("Upgrade")
	if upgrade == "websocket" || upgrade == "Websocket" {
		websocket.Handler(func(ws *websocket.Conn) {
			//Override default Read/Write timeout with sane value for a web socket request
			if err := ws.SetDeadline(time.Now().Add(time.Hour * 24)); err != nil {
				ERROR.Println("SetDeadLine failed:", err)
			}
			r.Method = "WS"
			handleInternal(w, r, ws)
		}).ServeHTTP(w, r)
	} else {
		handleInternal(w, r, nil)
	}
}

func handleInternal(w http.ResponseWriter, r *http.Request, ws *websocket.Conn) {
	// TODO For now this okay to put logger here for all the requests
	// However, it's best to have logging handler at server entry level
	start := time.Now()
	clientIP := ClientIP(r)

	var (
		req  = NewRequest(r)
		resp = NewResponse(w)
		c    = NewController(req, resp)
	)
	req.Websocket = ws
	c.ClientIP = clientIP

	Filters[0](c, Filters[1:])
	if c.Result != nil {
		c.Result.Apply(req, resp)
	} else if c.Response.Status != 0 {
		c.Response.Out.WriteHeader(c.Response.Status)
	}
	// Close the Writer if we can
	if w, ok := resp.Out.(io.Closer); ok {
		_ = w.Close()
	}

	// Revel request access log format
	// RequestStartTime ClientIP ResponseStatus RequestLatency HTTPMethod URLPath
	// Sample format:
	// 2016/05/25 17:46:37.112 127.0.0.1 200  270.157µs GET /
	requestLog.Printf("%v %v %v %10v %v %v",
		start.Format(requestLogTimeFormat),
		clientIP,
		c.Response.Status,
		time.Since(start),
		r.Method,
		r.URL.Path,
	)
}

// InitServer intializes the server and returns the handler
// It can be used as an alternative entry-point if one needs the http handler
// to be exposed. E.g. to run on multiple addresses and ports or to set custom
// TLS options.
func InitServer() http.HandlerFunc {
	runStartupHooks()

	// Load templates
	MainTemplateLoader = NewTemplateLoader(TemplatePaths)
	if err := MainTemplateLoader.Refresh(); err != nil {
		ERROR.Println(err)
	}

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
	}

	return http.HandlerFunc(handle)
}

// Run the server.
// This is called from the generated main file.
// If port is non-zero, use that.  Else, read the port from app.conf.
func Run(port int) {
	address := HTTPAddr
	if port == 0 {
		port = HTTPPort
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

	Server = &http.Server{
		Addr:         localAddress,
		Handler:      http.HandlerFunc(handle),
		ReadTimeout:  time.Duration(Config.IntDefault("http.timeout.read", 0)) * time.Second,
		WriteTimeout: time.Duration(Config.IntDefault("http.timeout.write", 0)) * time.Second,
	}

	InitServer()

	go func() {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Listening on %s...\n", Server.Addr)
	}()

	if HTTPSsl {
		if network != "tcp" {
			// This limitation is just to reduce complexity, since it is standard
			// to terminate SSL upstream when using unix domain sockets.
			ERROR.Fatalln("SSL is only supported for TCP sockets. Specify a port to listen on.")
		}
		ERROR.Fatalln("Failed to listen:",
			Server.ListenAndServeTLS(HTTPSslCert, HTTPSslKey))
	} else {
		listener, err := net.Listen(network, Server.Addr)
		if err != nil {
			ERROR.Fatalln("Failed to listen:", err)
		}
		ERROR.Fatalln("Failed to serve:", Server.Serve(listener))
	}
}

func runStartupHooks() {
	sort.Sort(startupHooks)
	for _, hook := range startupHooks {
		hook.f()
	}
}

type StartupHook struct {
	order int
	f     func()
}

type StartupHooks []StartupHook

var startupHooks StartupHooks

func (slice StartupHooks) Len() int {
	return len(slice)
}

func (slice StartupHooks) Less(i, j int) bool {
	return slice[i].order < slice[j].order
}

func (slice StartupHooks) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// OnAppStart registers a function to be run at app startup.
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
func OnAppStart(f func(), order ...int) {
	o := 1
	if len(order) > 0 {
		o = order[0]
	}
	startupHooks = append(startupHooks, StartupHook{order: o, f: f})
}
