// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"fmt"
	"github.com/revel/revel/session"
	"os"
	"strconv"
	"strings"
	"github.com/revel/revel/utils"
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
	CurrentEngine.Init(ServerEngineInit)
	initControllerStack()
	startupHooks.Run()

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
	defer func() {
		if r := recover(); r != nil {
			RevelLog.Crit("Recovered error in startup", "error", r)
			RaiseEvent(REVEL_FAILURE, r)
			panic("Fatal error in startup")
		}
	}()

	// Initialize the session logger, must be initiated from this app to avoid
	// circular references
	session.InitSession(RevelLog)

	// Create the CurrentEngine instance from the application config
	InitServerEngine(port, Config.StringDefault("server.engine", GO_NATIVE_SERVER_ENGINE))
	RaiseEvent(ENGINE_BEFORE_INITIALIZED, nil)
	InitServer()
	RaiseEvent(ENGINE_STARTED, nil)
	// This is needed for the harness to recognize that the server is started, it looks for the word
	// "Listening" in the stdout stream

	fmt.Fprintf(os.Stdout, "Revel engine is listening on.. %s\n", ServerEngineInit.Address)
	// Start never returns,
	CurrentEngine.Start()
	fmt.Fprintf(os.Stdout, "Revel engine is NOT listening on.. %s\n", ServerEngineInit.Address)
	RaiseEvent(ENGINE_SHUTDOWN, nil)
	shutdownHooks.Run()
	println("\nRevel exited normally\n")
}

// Build an engine initialization object and start the engine
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
	}
	AddInitEventHandler(CurrentEngine.Event)
}

// Initialize the controller stack for the application
func initControllerStack() {
	RevelConfig.Controller.Reuse = Config.BoolDefault("revel.controller.reuse",true)

	if RevelConfig.Controller.Reuse {
		RevelConfig.Controller.Stack = utils.NewStackLock(
			Config.IntDefault("revel.controller.stack", 10),
			Config.IntDefault("revel.controller.maxstack", 200), func() interface{} {
				return NewControllerEmpty()
			})
		RevelConfig.Controller.CachedStackSize = Config.IntDefault("revel.cache.controller.stack", 10)
		RevelConfig.Controller.CachedStackMaxSize = Config.IntDefault("revel.cache.controller.maxstack", 100)
		RevelConfig.Controller.CachedMap = map[string]*utils.SimpleLockStack{}
	}
}

// Called to stop the server
func StopServer(value interface{}) EventResponse {
	return RaiseEvent(ENGINE_SHUTDOWN_REQUEST,value)
}