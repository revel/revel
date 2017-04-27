// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
    "io"
    "mime/multipart"
    "net/url"
    "time"
    "errors"
)
const (
    /* Minimum Engine Type Values */
    _ = iota
    ENGINE_RESPONSE_STATUS
    ENGINE_WRITER
    ENGINE_PARAMETERS
    ENGINE_PATH
    ENGINE_REQUEST
    ENGINE_RESPONSE

)
const (
    /* HTTP Engine Type Values Starts at 1000 */
    HTTP_QUERY = ENGINE_PARAMETERS
    HTTP_PATH  = ENGINE_PATH
    HTTP_FORM  = iota + 1000
    HTTP_MULTIPART_FORM = iota + 1000
    HTTP_METHOD = iota + 1000
    HTTP_REQUEST_URI = iota + 1000
    HTTP_REMOTE_ADDR = iota + 1000
    HTTP_HOST = iota + 1000
    HTTP_SERVER_HEADER = iota + 1000
    HTTP_STREAM_WRITER = iota + 1000
    HTTP_WRITER = ENGINE_WRITER
)
type (
    ServerContext interface {
        GetRequest()  ServerRequest
        GetResponse() ServerResponse
    }


    // Callback ServerRequest type
    ServerRequest interface {
        GetRaw() interface{}
        Get(theType int) (interface{}, error)
        Set(theType int, theValue interface{}) bool
    }
    // Callback ServerResponse type
    ServerResponse interface {
        GetRaw() interface{}
        Get(theType int) (interface{}, error)
        Set(theType int, theValue interface{}) bool
    }
    // Callback WebSocket type
    ServerWebSocket interface {
        ServerResponse
        MessageSendJson(v interface{}) error
        MessageReceiveJson(v interface{}) error
    }

    // Expected response for HTTP_SERVER_HEADER type (if implemented)
    ServerHeader interface {
        SetCookie(cookie string)
        GetCookie(key string) (value ServerCookie, err error)
        Set(key string, value string)
        Add(key string, value string)
        Del(key string)
        Get(key string) (value string)
        SetStatus(statusCode int)
    }

    // Expected response for FROM_HTTP_COOKIE type (if implemented)
    ServerCookie interface {
        GetValue() string
    }

    // Expected response for HTTP_MULTIPART_FORM
    ServerMultipartForm interface {
        GetFile() map[string][]*multipart.FileHeader
        GetValue() url.Values
        RemoveAll() error
    }
    StreamWriter interface {
        WriteStream(name string,contentlen int64, modtime time.Time, reader io.Reader) error
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
    Callback func(ServerContext)
}
type ServerEngineEmpty struct {

}

var (
    // The simple stacks for response and controllers are a linked list
    // of reused objects. 
    controllerStack           *SimpleLockStack
    cachedControllerMap       = map[string]*SimpleLockStack{}
    cachedControllerStackSize = 10
    cachedControllerStackMaxSize = 10
)

func handleInternal(ctx ServerContext) {
    start := time.Now()

    var (
        c = controllerStack.Pop().(*Controller)
        req,resp = c.Request, c.Response
    )
    c.SetController(ctx)
    req.Websocket, _ = ctx.GetResponse().(ServerWebSocket)

    clientIP := ClientIP(req)


    // Once finished in the internal, we can return these to the stack
    defer func() {
        controllerStack.Push(c)
    }()

    c.ClientIP = clientIP

    Filters[0](c, Filters[1:])
    if c.Result != nil {
        c.Result.Apply(req, resp)
    } else if c.Response.Status != 0 {
        c.Response.SetStatus(c.Response.Status)
    }
    // Close the Writer if we can
    if w, ok := resp.GetWriter().(io.Closer); ok {
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
        req.Method,
        req.GetPath(),
    )
}

var (
    ENGINE_UNKNOWN_GET = errors.New("Server Engine Invalid Get")
)

func (e *ServerEngineEmpty) Get(_ string) interface{} {
    return nil
}
func (e *ServerEngineEmpty) Set(_ string, _ interface{}) bool {
    return false
}

