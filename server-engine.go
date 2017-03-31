// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)


type ServerEngine interface {
    // Initialize the server (non blocking)
    Init(init *EngineInit)
    // Starts the server. This will block until server is stopped
    Start()
    // Fires a new event to the server
    Event(event string, args interface{})
    // Returns the engine instance for specific calls
    Engine() interface{}
    // Returns the engine Name
    Name() string
    // Handle the request an response
    Handle(w http.ResponseWriter, r *http.Request)
}
type EngineInit struct {
        Address,
        Network string
        Port int
        Callback func(http.ResponseWriter, *http.Request, *websocket.Conn)
}

// Register the GOHttpServer engine
func init() {
    RegisterServerEngine(&GOHttpServer{})
}
type GOHttpServer struct {
    Server *http.Server
    ServerInit *EngineInit
}

func (g *GOHttpServer) Init(init *EngineInit) {
    g.ServerInit = init
	g.Server = &http.Server{
		Addr:         init.Address,
		Handler:      http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request){
            g.Handle(writer,request)
        }),
		ReadTimeout:  time.Duration(Config.IntDefault("http.timeout.read", 0)) * time.Second,
		WriteTimeout: time.Duration(Config.IntDefault("http.timeout.write", 0)) * time.Second,
	}
    // Server already initialized


}
func (g *GOHttpServer) Start() {
	go func() {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Listening on %s...\n", g.Server.Addr)
	}()
	if HTTPSsl {
		if g.ServerInit.Network != "tcp" {
			// This limitation is just to reduce complexity, since it is standard
			// to terminate SSL upstream when using unix domain sockets.
			ERROR.Fatalln("SSL is only supported for TCP sockets. Specify a port to listen on.")
		}
		ERROR.Fatalln("Failed to listen:",
			g.Server.ListenAndServeTLS(HTTPSslCert, HTTPSslKey))
	} else {
		listener, err := net.Listen(g.ServerInit.Network, g.Server.Addr)
		if err != nil {
			ERROR.Fatalln("Failed to listen:", err)
		}
		ERROR.Fatalln("Failed to serve:", g.Server.Serve(listener))
	}

}

func (g *GOHttpServer) Handle(w http.ResponseWriter, r *http.Request) {
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
			g.ServerInit.Callback(w, r, ws)
		}).ServeHTTP(w, r)
	} else {
		g.ServerInit.Callback(w, r, nil)
	}
}

const GO_NATIVE_SERVER_ENGINE = "go"

func (g *GOHttpServer) Name() string {
    return GO_NATIVE_SERVER_ENGINE
}

func (g *GOHttpServer) Engine() interface{} {
    return g.Server
}

func (g *GOHttpServer) Event(event string, args interface{}) {

}
