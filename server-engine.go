// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"net/http"

	"golang.org/x/net/websocket"
)


type ServerEngine interface {
    // Initialize the server (non blocking)
    Init(init *EngineInit)
    // Starts the server. This will block until server is stopped
    Start()
    // Fires a new event to the server
    Event(event int, args interface{})
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
