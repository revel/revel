// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"errors"
	"io"
	"mime/multipart"
	"net/url"
	"time"
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
	HTTP_QUERY          = ENGINE_PARAMETERS
	HTTP_PATH           = ENGINE_PATH
	HTTP_BODY           = iota + 1000
	HTTP_FORM           = iota + 1000
	HTTP_MULTIPART_FORM = iota + 1000
	HTTP_METHOD         = iota + 1000
	HTTP_REQUEST_URI    = iota + 1000
	HTTP_REMOTE_ADDR    = iota + 1000
	HTTP_HOST           = iota + 1000
	HTTP_URL            = iota + 1000
	HTTP_SERVER_HEADER  = iota + 1000
	HTTP_STREAM_WRITER  = iota + 1000
	HTTP_WRITER         = ENGINE_WRITER
)

type (
	ServerContext interface {
		GetRequest() ServerRequest
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
		ServerRequest
	}
	// Callback WebSocket type
	ServerWebSocket interface {
		ServerResponse
		MessageSendJSON(v interface{}) error
		MessageReceiveJSON(v interface{}) error
	}

	// Expected response for HTTP_SERVER_HEADER type (if implemented)
	ServerHeader interface {
		SetCookie(cookie string)
		GetCookie(key string) (value ServerCookie, err error)
		Set(key string, value string)
		Add(key string, value string)
		Del(key string)
		Get(key string) (value []string)
		SetStatus(statusCode int)
	}

	// Expected response for FROM_HTTP_COOKIE type (if implemented)
	ServerCookie interface {
		GetValue() string
	}

	// Expected response for HTTP_MULTIPART_FORM
	ServerMultipartForm interface {
		GetFiles() map[string][]*multipart.FileHeader
		GetValues() url.Values
		RemoveAll() error
	}
	StreamWriter interface {
		WriteStream(name string, contentlen int64, modtime time.Time, reader io.Reader) error
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
	controllerStack              *SimpleLockStack
	cachedControllerMap          = map[string]*SimpleLockStack{}
	cachedControllerStackSize    = 10
	cachedControllerStackMaxSize = 10
)

func handleInternal(ctx ServerContext) {
	start := time.Now()

	var (
		c         = controllerStack.Pop().(*Controller)
		req, resp = c.Request, c.Response
	)
	c.SetController(ctx)
	req.WebSocket, _ = ctx.GetResponse().(ServerWebSocket)

	clientIP := ClientIP(req)

	// Once finished in the internal, we can return these to the stack
	defer func() {
		controllerStack.Push(c)
	}()

	c.ClientIP = clientIP
	c.Log = AppLog.New("ip", clientIP,
		"path", req.GetPath(), "method", req.Method)
	// Call the first filter, this will process the request
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
	// Sample format: terminal format
	// INFO 2017/08/02 22:31:41 server-engine.go:168: Request Stats                            ip=::1 path=/public/img/favicon.png method=GET action=Static.Serve namespace=static\\ start=2017/08/02 22:31:41 status=200 duration_seconds=0.0007656
	// Recommended storing format to json code which looks like
	// {"action":"Static.Serve","caller":"server-engine.go:168","duration_seconds":0.00058336,"ip":"::1","lvl":3,
	// "method":"GET","msg":"Request Stats","namespace":"static\\","path":"/public/img/favicon.png",
	// "start":"2017-08-02T22:34:08-0700","status":200,"t":"2017-08-02T22:34:08.303112145-07:00"}

	c.Log.Info("Request Stats",
		"start", start,
		"status", c.Response.Status,
		"duration_seconds", time.Since(start).Seconds(), "section", "requestlog",
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
