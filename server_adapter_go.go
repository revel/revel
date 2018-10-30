package revel

import (
	"net"
	"net/http"
	"time"
	"context"
	"golang.org/x/net/websocket"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"os/signal"
	"path"
	"sort"
	"strconv"
	"strings"
	"github.com/revel/revel/utils"
)

// Register the GoHttpServer engine
func init() {
	AddInitEventHandler(func(typeOf Event, value interface{}) (responseOf EventResponse) {
		if typeOf == REVEL_BEFORE_MODULES_LOADED {
			RegisterServerEngine(GO_NATIVE_SERVER_ENGINE, func() ServerEngine { return &GoHttpServer{} })
		}
		return
	})
}

// The Go HTTP server
type GoHttpServer struct {
	Server               *http.Server           // The server instance
	ServerInit           *EngineInit            // The server engine initialization
	MaxMultipartSize     int64                  // The largest size of file to accept
	goContextStack       *utils.SimpleLockStack // The context stack Set via server.context.stack, server.context.maxstack
	goMultipartFormStack *utils.SimpleLockStack // The multipart form stack set via server.form.stack, server.form.maxstack
	HttpMuxList          ServerMuxList
	HasAppMux            bool
	signalChan           chan os.Signal
}

// Called to initialize the server with this EngineInit
func (g *GoHttpServer) Init(init *EngineInit) {
	g.MaxMultipartSize = int64(Config.IntDefault("server.request.max.multipart.filesize", 32)) << 20 /* 32 MB */
	g.goContextStack = utils.NewStackLock(Config.IntDefault("server.context.stack", 100),
		Config.IntDefault("server.context.maxstack", 200),
		func() interface{} {
			return NewGoContext(g)
		})
	g.goMultipartFormStack = utils.NewStackLock(Config.IntDefault("server.form.stack", 100),
		Config.IntDefault("server.form.maxstack", 200),
		func() interface{} { return &GoMultipartForm{} })
	g.ServerInit = init

	revelHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		g.Handle(writer, request)
	})

	// Adds the mux list
	g.HttpMuxList = init.HTTPMuxList
	sort.Sort(g.HttpMuxList)
	g.HasAppMux = len(g.HttpMuxList) > 0
	g.signalChan = make(chan os.Signal)

	g.Server = &http.Server{
		Addr:         init.Address,
		Handler:      revelHandler,
		ReadTimeout:  time.Duration(Config.IntDefault("http.timeout.read", 0)) * time.Second,
		WriteTimeout: time.Duration(Config.IntDefault("http.timeout.write", 0)) * time.Second,
	}

}

// Handler is assigned in the Init
func (g *GoHttpServer) Start() {
	go func() {
		time.Sleep(100 * time.Millisecond)
		serverLogger.Debugf("Start: Listening on %s...", g.Server.Addr)
	}()
	if HTTPSsl {
		if g.ServerInit.Network != "tcp" {
			// This limitation is just to reduce complexity, since it is standard
			// to terminate SSL upstream when using unix domain sockets.
			serverLogger.Fatal("SSL is only supported for TCP sockets. Specify a port to listen on.")
		}
		serverLogger.Fatal("Failed to listen:", "error",
			g.Server.ListenAndServeTLS(HTTPSslCert, HTTPSslKey))
	} else {
		listener, err := net.Listen(g.ServerInit.Network, g.Server.Addr)
		if err != nil {
			serverLogger.Fatal("Failed to listen:", "error", err)
		}
		serverLogger.Warn("Server exiting:", "error", g.Server.Serve(listener))
	}
}

// Handle the request and response for the server
func (g *GoHttpServer) Handle(w http.ResponseWriter, r *http.Request) {
	// This section is called if the developer has added custom mux to the app
	if g.HasAppMux && g.handleAppMux(w, r) {
		return
	}
	g.handleMux(w, r)
}

// Handle the request and response for the servers mux
func (g *GoHttpServer) handleAppMux(w http.ResponseWriter, r *http.Request) bool {
	// Check the prefix and split them
	requestPath := path.Clean(r.URL.Path)
	if handler, hasHandler := g.HttpMuxList.Find(requestPath); hasHandler {
		clientIP := HttpClientIP(r)
		localLog := AppLog.New("ip", clientIP,
			"path", r.URL.Path, "method", r.Method)
		defer func() {
			if err := recover(); err != nil {
				localLog.Error("An error was caught using the handler", "path", requestPath, "error", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		start := time.Now()
		handler.(http.HandlerFunc)(w, r)
		localLog.Info("Request Stats",
			"start", start,
			"duration_seconds", time.Since(start).Seconds(), "section", "requestlog",
		)
		return true
	}
	return false
}

// Passes the server request to Revel
func (g *GoHttpServer) handleMux(w http.ResponseWriter, r *http.Request) {
	if maxRequestSize := int64(Config.IntDefault("http.maxrequestsize", 0)); maxRequestSize > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	}

	upgrade := r.Header.Get("Upgrade")
	context := g.goContextStack.Pop().(*GoContext)
	defer func() {
		g.goContextStack.Push(context)
	}()
	context.Request.SetRequest(r)
	context.Response.SetResponse(w)

	if upgrade == "websocket" || upgrade == "Websocket" {
		websocket.Handler(func(ws *websocket.Conn) {
			//Override default Read/Write timeout with sane value for a web socket request
			if err := ws.SetDeadline(time.Now().Add(time.Hour * 24)); err != nil {
				serverLogger.Error("SetDeadLine failed:", err)
			}
			r.Method = "WS"
			context.Request.WebSocket = ws
			context.WebSocket = &GoWebSocket{Conn: ws, GoResponse: *context.Response}
			g.ServerInit.Callback(context)
		}).ServeHTTP(w, r)
	} else {
		g.ServerInit.Callback(context)
	}
}

// ClientIP method returns client IP address from HTTP request.
//
// Note: Set property "app.behind.proxy" to true only if Revel is running
// behind proxy like nginx, haproxy, apache, etc. Otherwise
// you may get inaccurate Client IP address. Revel parses the
// IP address in the order of X-Forwarded-For, X-Real-IP.
//
// By default revel will get http.Request's RemoteAddr
func HttpClientIP(r *http.Request) string {
	if Config.BoolDefault("app.behind.proxy", false) {
		// Header X-Forwarded-For
		if fwdFor := strings.TrimSpace(r.Header.Get(HdrForwardedFor)); fwdFor != "" {
			index := strings.Index(fwdFor, ",")
			if index == -1 {
				return fwdFor
			}
			return fwdFor[:index]
		}

		// Header X-Real-Ip
		if realIP := strings.TrimSpace(r.Header.Get(HdrRealIP)); realIP != "" {
			return realIP
		}
	}

	if remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return remoteAddr
	}

	return ""
}

// The server key name
const GO_NATIVE_SERVER_ENGINE = "go"

// Returns the name of this engine
func (g *GoHttpServer) Name() string {
	return GO_NATIVE_SERVER_ENGINE
}

// Returns stats for this engine
func (g *GoHttpServer) Stats() map[string]interface{} {
	return map[string]interface{}{
		"Go Engine Context": g.goContextStack.String(),
		"Go Engine Forms":   g.goMultipartFormStack.String(),
	}
}

// Return the engine instance
func (g *GoHttpServer) Engine() interface{} {
	return g.Server
}

// Handles an event from Revel
func (g *GoHttpServer) Event(event Event, args interface{}) (r EventResponse) {
	switch event {
	case ENGINE_STARTED:
		signal.Notify(g.signalChan, os.Interrupt, os.Kill)
		go func() {
			_ = <-g.signalChan
			serverLogger.Info("Received quit singal Please wait ... ")
			RaiseEvent(ENGINE_SHUTDOWN_REQUEST, nil)
		}()
	case ENGINE_SHUTDOWN_REQUEST:
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(Config.IntDefault("app.cancel.timeout", 60)))
		defer cancel()
		g.Server.Shutdown(ctx)
	default:

	}

	return
}

type (
	// The go context
	GoContext struct {
		Request   *GoRequest   // The request
		Response  *GoResponse  // The response
		WebSocket *GoWebSocket // The websocket
	}

	// The go request
	GoRequest struct {
		Original        *http.Request    // The original
		FormParsed      bool             // True if form parsed
		MultiFormParsed bool             // True if multipart form parsed
		WebSocket       *websocket.Conn  // The websocket
		ParsedForm      *GoMultipartForm // The parsed form data
		Goheader        *GoHeader        // The header
		Engine          *GoHttpServer    // THe server
	}

	// The response
	GoResponse struct {
		Original http.ResponseWriter // The original writer
		Goheader *GoHeader           // The header
		Writer   io.Writer           // The writer
		Request  *GoRequest          // The request
		Engine   *GoHttpServer       // The engine
	}

	// The multipart form
	GoMultipartForm struct {
		Form *multipart.Form // The form
	}

	// The go header
	GoHeader struct {
		Source     interface{} // The source
		isResponse bool        // True if response header
	}

	// The websocket
	GoWebSocket struct {
		Conn       *websocket.Conn // The connection
		GoResponse                 // The response
	}

	// The cookie
	GoCookie http.Cookie
)

// Create a new go context
func NewGoContext(instance *GoHttpServer) *GoContext {
	// This bit in here is for the test cases, which pass in a nil value
	if instance == nil {
		instance = &GoHttpServer{MaxMultipartSize: 32 << 20}
		instance.goContextStack = utils.NewStackLock(100, 200,
			func() interface{} {
				return NewGoContext(instance)
			})
		instance.goMultipartFormStack = utils.NewStackLock(100, 200,
			func() interface{} { return &GoMultipartForm{} })
	}
	c := &GoContext{Request: &GoRequest{Goheader: &GoHeader{}, Engine: instance}}
	c.Response = &GoResponse{Goheader: &GoHeader{}, Request: c.Request, Engine: instance}
	return c
}

// get the request
func (c *GoContext) GetRequest() ServerRequest {
	return c.Request
}

// Get the response
func (c *GoContext) GetResponse() ServerResponse {
	if c.WebSocket != nil {
		return c.WebSocket
	}
	return c.Response
}

// Destroy the context
func (c *GoContext) Destroy() {
	c.Response.Destroy()
	c.Request.Destroy()
	if c.WebSocket != nil {
		c.WebSocket.Destroy()
		c.WebSocket = nil
	}
}

// Communicate with the server engine to exchange data
func (r *GoRequest) Get(key int) (value interface{}, err error) {
	switch key {
	case HTTP_SERVER_HEADER:
		value = r.GetHeader()
	case HTTP_MULTIPART_FORM:
		value, err = r.GetMultipartForm()
	case HTTP_QUERY:
		value = r.Original.URL.Query()
	case HTTP_FORM:
		value, err = r.GetForm()
	case HTTP_REQUEST_URI:
		value = r.Original.URL.String()
	case HTTP_REQUEST_CONTEXT:
		value = r.Original.Context()
	case HTTP_REMOTE_ADDR:
		value = r.Original.RemoteAddr
	case HTTP_METHOD:
		value = r.Original.Method
	case HTTP_PATH:
		value = r.Original.URL.Path
	case HTTP_HOST:
		value = r.Original.Host
	case HTTP_URL:
		value = r.Original.URL
	case HTTP_BODY:
		value = r.Original.Body
	default:
		err = ENGINE_UNKNOWN_GET
	}

	return
}

// Sets the request key with value
func (r *GoRequest) Set(key int, value interface{}) bool {
	return false
}

// Returns the form
func (r *GoRequest) GetForm() (url.Values, error) {
	if !r.FormParsed {
		if e := r.Original.ParseForm(); e != nil {
			return nil, e
		}
		r.FormParsed = true
	}

	return r.Original.Form, nil
}

// Returns the form
func (r *GoRequest) GetMultipartForm() (ServerMultipartForm, error) {
	if !r.MultiFormParsed {
		if e := r.Original.ParseMultipartForm(r.Engine.MaxMultipartSize); e != nil {
			return nil, e
		}
		r.ParsedForm = r.Engine.goMultipartFormStack.Pop().(*GoMultipartForm)
		r.ParsedForm.Form = r.Original.MultipartForm
	}

	return r.ParsedForm, nil
}

// Returns the header
func (r *GoRequest) GetHeader() ServerHeader {
	return r.Goheader
}

// Returns the raw value
func (r *GoRequest) GetRaw() interface{} {
	return r.Original
}

// Sets the request
func (r *GoRequest) SetRequest(req *http.Request) {
	r.Original = req
	r.Goheader.Source = r
	r.Goheader.isResponse = false

}

// Destroy the request
func (r *GoRequest) Destroy() {
	r.Goheader.Source = nil
	r.Original = nil
	r.FormParsed = false
	r.MultiFormParsed = false
	r.ParsedForm = nil
}

// Gets the key from the response
func (r *GoResponse) Get(key int) (value interface{}, err error) {
	switch key {
	case HTTP_SERVER_HEADER:
		value = r.Header()
	case HTTP_STREAM_WRITER:
		value = r
	case HTTP_WRITER:
		value = r.Writer
	default:
		err = ENGINE_UNKNOWN_GET
	}
	return
}

// Sets the key with the value
func (r *GoResponse) Set(key int, value interface{}) (set bool) {
	switch key {
	case ENGINE_RESPONSE_STATUS:
		r.Header().SetStatus(value.(int))
		set = true
	case HTTP_WRITER:
		r.SetWriter(value.(io.Writer))
		set = true
	}
	return
}

// Sets the header
func (r *GoResponse) Header() ServerHeader {
	return r.Goheader
}

// Gets the original response
func (r *GoResponse) GetRaw() interface{} {
	return r.Original
}

// Sets the writer
func (r *GoResponse) SetWriter(writer io.Writer) {
	r.Writer = writer
}

// Write output to stream
func (r *GoResponse) WriteStream(name string, contentlen int64, modtime time.Time, reader io.Reader) error {
	// Check to see if the output stream is modified, if not send it using the
	// Native writer
	written := false
	if _, ok := r.Writer.(http.ResponseWriter); ok {
		if rs, ok := reader.(io.ReadSeeker); ok {
			http.ServeContent(r.Original, r.Request.Original, name, modtime, rs)
			written = true
		}
	}
	if !written {
		// Else, do a simple io.Copy.
		ius := r.Request.Original.Header.Get("If-Unmodified-Since")
		if t, err := http.ParseTime(ius); err == nil && !modtime.IsZero() {
			// The Date-Modified header truncates sub-second precision, so
			// use mtime < t+1s instead of mtime <= t to check for unmodified.
			if modtime.Before(t.Add(1 * time.Second)) {
				h := r.Original.Header()
				delete(h, "Content-Type")
				delete(h, "Content-Length")
				if h.Get("Etag") != "" {
					delete(h, "Last-Modified")
				}
				r.Original.WriteHeader(http.StatusNotModified)
				return nil
			}
		}

		if contentlen != -1 {
			header := ServerHeader(r.Goheader)
			if writer, found := r.Writer.(*CompressResponseWriter); found {
				header = ServerHeader(writer.Header)
			}
			header.Set("Content-Length", strconv.FormatInt(contentlen, 10))
		}
		if _, err := io.Copy(r.Writer, reader); err != nil {
			r.Original.WriteHeader(http.StatusInternalServerError)
			return err
		} else {
			r.Original.WriteHeader(http.StatusOK)
		}
	}
	return nil
}

// Frees response
func (r *GoResponse) Destroy() {
	if c, ok := r.Writer.(io.Closer); ok {
		c.Close()
	}
	r.Goheader.Source = nil
	r.Original = nil
	r.Writer = nil
}

// Sets the response
func (r *GoResponse) SetResponse(w http.ResponseWriter) {
	r.Original = w
	r.Writer = w
	r.Goheader.Source = r
	r.Goheader.isResponse = true

}

// Sets the cookie
func (r *GoHeader) SetCookie(cookie string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Add("Set-Cookie", cookie)
	}
}

// Gets the cookie
func (r *GoHeader) GetCookie(key string) (value ServerCookie, err error) {
	if !r.isResponse {
		var cookie *http.Cookie
		if cookie, err = r.Source.(*GoRequest).Original.Cookie(key); err == nil {
			value = GoCookie(*cookie)

		}

	}
	return
}

// Sets (replaces) header key
func (r *GoHeader) Set(key string, value string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Set(key, value)
	}
}

// Adds the header key
func (r *GoHeader) Add(key string, value string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Add(key, value)
	}
}

// Deletes the header key
func (r *GoHeader) Del(key string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Del(key)
	}
}

// Gets the header key
func (r *GoHeader) Get(key string) (value []string) {
	if !r.isResponse {
		value = r.Source.(*GoRequest).Original.Header[key]
		if len(value) == 0 {
			if ihead := r.Source.(*GoRequest).Original.Header.Get(key); ihead != "" {
				value = append(value, ihead)
			}
		}
	} else {
		value = r.Source.(*GoResponse).Original.Header()[key]
	}
	return
}

// Returns list of header keys
func (r *GoHeader) GetKeys() (value []string) {
	if !r.isResponse {
		for key := range r.Source.(*GoRequest).Original.Header {
			value = append(value, key)
		}
	} else {
		for key := range r.Source.(*GoResponse).Original.Header() {
			value = append(value, key)
		}
	}
	return
}

// Sets the status of the header
func (r *GoHeader) SetStatus(statusCode int) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.WriteHeader(statusCode)
	}
}

// Return cookies value
func (r GoCookie) GetValue() string {
	return r.Value
}

// Return files from the form
func (f *GoMultipartForm) GetFiles() map[string][]*multipart.FileHeader {
	return f.Form.File
}

// Return values from the form
func (f *GoMultipartForm) GetValues() url.Values {
	return url.Values(f.Form.Value)
}

// Remove all values from the form freeing memory
func (f *GoMultipartForm) RemoveAll() error {
	return f.Form.RemoveAll()
}

/**
 * Message send JSON
 */
func (g *GoWebSocket) MessageSendJSON(v interface{}) error {
	return websocket.JSON.Send(g.Conn, v)
}

/**
 * Message receive JSON
 */
func (g *GoWebSocket) MessageReceiveJSON(v interface{}) error {
	return websocket.JSON.Receive(g.Conn, v)
}

/**
 * Message Send
 */
func (g *GoWebSocket) MessageSend(v interface{}) error {
	return websocket.Message.Send(g.Conn, v)
}

/**
 * Message receive
 */
func (g *GoWebSocket) MessageReceive(v interface{}) error {
	return websocket.Message.Receive(g.Conn, v)
}
