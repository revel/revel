// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"sort"
	"strconv"
	"strings"
	"fmt"
	"os"
)

// Revel's variables server, router, etc
var (
	MainRouter         *Router
	MainTemplateLoader *TemplateLoader
	MainWatcher        *Watcher
	serverEngineMap    = map[string]func() ServerEngine{}
	CurrentEngine      ServerEngine
	ServerEngineInit   *EngineInit
	serverLogger       = RevelLog.New("section", "server")
)

func RegisterServerEngine(name string, loader func() ServerEngine) {
	serverLogger.Debug("RegisterServerEngine: Registered engine ", "name", name)
	serverEngineMap[name] = loader
}

// InitServer initializes the server and returns the handler
// It can be used as an alternative entry-point if one needs the http handler
// to be exposed. E.g. to run on multiple addresses and ports or to set custom
// TLS options.
func InitServer() {
	initControllerStack()
	runStartupHooks()

	// Load templates
	MainTemplateLoader = NewTemplateLoader(TemplatePaths)
	if err := MainTemplateLoader.Refresh(); err != nil {
		serverLogger.Debug("InitServer: Main template loader failed to refresh", "error", err)
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

}

// Run the server.
// This is called from the generated main file.
// If port is non-zero, use that.  Else, read the port from app.conf.
func Run(port int) {

	// Create the CurrentEngine instance from the application config
	InitServerEngine(port, Config.StringDefault("server.engine", GO_NATIVE_SERVER_ENGINE))
	CurrentEngine.Event(ENGINE_BEFORE_INITIALIZED, nil)
	fireEvent(ENGINE_BEFORE_INITIALIZED, nil)
	InitServer()
	fireEvent(ENGINE_STARTED, nil)
	CurrentEngine.Event(ENGINE_STARTED, nil)
	// This is needed for the harness to recognize that the server is started, it looks for the word
	// "Listening" in the stdout stream
	fmt.Fprintf(os.Stdout,"Listening on.. %s\n", ServerEngineInit.Address)
	CurrentEngine.Start()
	CurrentEngine.Event(ENGINE_SHUTDOWN, nil)
}

func InitServerEngine(port int, serverEngine string) {
	address := HTTPAddr
	if address == "" {
		address = "localhost"
	}
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

	if engineLoader, ok := serverEngineMap[serverEngine]; !ok {
		panic("Server Engine " + serverEngine + " Not found")
	} else {
		CurrentEngine = engineLoader()
		serverLogger.Debug("InitServerEngine: Found server engine and invoking", "name", CurrentEngine.Name())
		ServerEngineInit = &EngineInit{
			Address:  localAddress,
			Network:  network,
			Port:     port,
			Callback: handleInternal,
		}
		CurrentEngine.Init(ServerEngineInit)
	}
}
func initControllerStack() {
	controllerStack = NewStackLock(
		Config.IntDefault("revel.controller.stack", 10),
		Config.IntDefault("revel.controller.maxstack", 200), func() interface{} { return NewControllerEmpty() })
	cachedControllerStackSize = Config.IntDefault("revel.cache.controller.stack", 10)
	cachedControllerStackMaxSize = Config.IntDefault("revel.cache.controller.maxstack", 100)
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
