// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
    "io"
    "mime/multipart"
    "net/url"
    "time"
)

type (
    ServerRequest interface {
        GetQuery() url.Values
        GetForm() (url.Values, error)
        GetMultipartForm(maxsize int64) (ServerMultipartForm, error)
        GetHeader() ServerHeader
        GetRaw() interface{}
        GetMethod() string
        GetPath() string
        GetRequestURI() string
        GetRemoteAddr() string
        GetHost() string
    }
    ServerResponse interface {
        GetWriter() io.Writer
        SetWriter(io.Writer)
        Header() ServerHeader
        GetRaw() interface{}
        WriteStream(name string,contentlen int64, modtime time.Time, reader io.Reader) error
    }
    ServerMultipartForm interface {
        GetFile() map[string][]*multipart.FileHeader
        GetValue() url.Values
        RemoveAll() error
    }
    ServerHeader interface {
        SetCookie(cookie string)
        GetCookie(key string) (value ServerCookie, err error)
        Set(key string, value string)
        Add(key string, value string)
        Del(key string)
        Get(key string) (value string)
        SetStatus(statusCode int)
    }
    ServerCookie interface {
        GetValue() string
    }
    ServerWebSocket interface {
        MessageSendJson(v interface{}) error
        MessageReceiveJson(v interface{}) error
    }
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
    // Returns any stats
    Stats() map[string]interface{}
}
type EngineInit struct {
    Address,
    Network string
    Port     int
    Callback func(ServerResponse, ServerRequest, ServerWebSocket)
}

var (
    // The simple stacks for response and controllers are a linked list
    // of reused objects. 
    requestStack              *SimpleLockStack
    responseStack             *SimpleLockStack
    controllerStack           *SimpleLockStack
    cachedControllerMap       = map[string]*SimpleLockStack{}
    cachedControllerStackSize = 10
)

func handleInternal(w ServerResponse, r ServerRequest, ws ServerWebSocket) {
    // TODO For now this okay to put logger here for all the requests
    // However, it's best to have logging handler at server entry level
    start := time.Now()
    clientIP := ClientIP(r)

    var (
        req, resp, c = requestStack.Pop().(*Request), responseStack.Pop().(*Response), controllerStack.Pop().(*Controller)
    )
    req.SetRequest(r)
    req.Websocket = ws
    resp.SetResponse(w)

    c.SetController(req, resp)

    // Once finished in the internal, we can return these to the stack
    defer func() {
        requestStack.Push(req)
        responseStack.Push(resp)
        controllerStack.Push(c)
    }()

    c.ClientIP = clientIP

    Filters[0](c, Filters[1:])
    if c.Result != nil {
        c.Result.Apply(req, resp)
    } else if c.Response.Status != 0 {
        c.Response.Out.Header().SetStatus(c.Response.Status)
    }
    // Close the Writer if we can
    if w, ok := resp.Out.GetWriter().(io.Closer); ok {
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
        r.GetMethod(),
        r.GetPath(),
    )
}
