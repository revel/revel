package revel

import (
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
	"io"
	"mime/multipart"
	"net/url"
	"strconv"
)

// Register the GoHttpServer engine

func init() {
	RegisterServerEngine(GO_NATIVE_SERVER_ENGINE, func() ServerEngine { return &GoHttpServer{} })
}

type GoHttpServer struct {
	Server               *http.Server
	ServerInit           *EngineInit
	MaxMultipartSize     int64
	goContextStack       *SimpleLockStack
	goMultipartFormStack *SimpleLockStack
}

func (g *GoHttpServer) Init(init *EngineInit) {
	g.MaxMultipartSize = int64(Config.IntDefault("server.request.max.multipart.filesize", 32)) << 20 /* 32 MB */
	g.goContextStack = NewStackLock(Config.IntDefault("server.context.stack", 100),
		Config.IntDefault("server.context.maxstack", 200),
		func() interface{} {
			return NewGoContext(g)
		})
	g.goMultipartFormStack = NewStackLock(Config.IntDefault("server.form.stack", 100),
		Config.IntDefault("server.form.maxstack", 200),
		func() interface{} { return &GoMultipartForm{} })
	g.ServerInit = init
	g.Server = &http.Server{
		Addr: init.Address,
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			g.Handle(writer, request)
		}),
		ReadTimeout:  time.Duration(Config.IntDefault("http.timeout.read", 0)) * time.Second,
		WriteTimeout: time.Duration(Config.IntDefault("http.timeout.write", 0)) * time.Second,
	}
	// Server already initialized

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
		serverLogger.Fatal("Failed to serve:", "error", g.Server.Serve(listener))
	}

}

func (g *GoHttpServer) Handle(w http.ResponseWriter, r *http.Request) {
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

const GO_NATIVE_SERVER_ENGINE = "go"

func (g *GoHttpServer) Name() string {
	return GO_NATIVE_SERVER_ENGINE
}

func (g *GoHttpServer) Stats() map[string]interface{} {
	return map[string]interface{}{
		"Go Engine Context": g.goContextStack.String(),
		"Go Engine Forms":   g.goMultipartFormStack.String(),
	}
}

func (g *GoHttpServer) Engine() interface{} {
	return g.Server
}

func (g *GoHttpServer) Event(event int, args interface{}) {

}

type (
	GoContext struct {
		Request   *GoRequest
		Response  *GoResponse
		WebSocket *GoWebSocket
	}
	GoRequest struct {
		Original        *http.Request
		FormParsed      bool
		MultiFormParsed bool
		WebSocket       *websocket.Conn
		ParsedForm      *GoMultipartForm
		Goheader        *GoHeader
		Engine          *GoHttpServer
	}

	GoResponse struct {
		Original http.ResponseWriter
		Goheader *GoHeader
		Writer   io.Writer
		Request  *GoRequest
		Engine   *GoHttpServer
	}
	GoMultipartForm struct {
		Form *multipart.Form
	}
	GoHeader struct {
		Source     interface{}
		isResponse bool
	}
	GoWebSocket struct {
		Conn *websocket.Conn
		GoResponse
	}
	GoCookie http.Cookie
)

func NewGoContext(instance *GoHttpServer) *GoContext {
	if instance == nil {
		instance = &GoHttpServer{MaxMultipartSize: 32 << 20}
		instance.goContextStack = NewStackLock(100, 200,
			func() interface{} {
				return NewGoContext(instance)
			})
		instance.goMultipartFormStack = NewStackLock(100, 200,
			func() interface{} { return &GoMultipartForm{} })
	}
	c := &GoContext{Request: &GoRequest{Goheader: &GoHeader{}, Engine: instance}}
	c.Response = &GoResponse{Goheader: &GoHeader{}, Request: c.Request, Engine: instance}
	return c
}
func (c *GoContext) GetRequest() ServerRequest {
	return c.Request
}
func (c *GoContext) GetResponse() ServerResponse {
	if c.WebSocket != nil {
		return c.WebSocket
	}
	return c.Response
}
func (c *GoContext) Destroy() {
	c.Response.Destroy()
	c.Request.Destroy()
	if c.WebSocket != nil {
		c.WebSocket.Destroy()
	}
}
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
func (r *GoRequest) Set(key int, value interface{}) bool {
	return false
}

func (r *GoRequest) GetForm() (url.Values, error) {
	if !r.FormParsed {
		if e := r.Original.ParseForm(); e != nil {
			return nil, e
		}
		r.FormParsed = true
	}

	return r.Original.Form, nil
}
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
func (r *GoRequest) GetHeader() ServerHeader {
	return r.Goheader
}
func (r *GoRequest) GetRaw() interface{} {
	return r.Original
}
func (r *GoRequest) SetRequest(req *http.Request) {
	r.Original = req
	r.Goheader.Source = r
	r.Goheader.isResponse = false

}
func (r *GoRequest) Destroy() {
	r.Goheader.Source = nil
	r.Original = nil
	r.FormParsed = false
	r.MultiFormParsed = false
	r.ParsedForm = nil
}
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

func (r *GoResponse) Header() ServerHeader {
	return r.Goheader
}
func (r *GoResponse) GetRaw() interface{} {
	return r.Original
}
func (r *GoResponse) SetWriter(writer io.Writer) {
	r.Writer = writer
}
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
			r.Original.Header().Set("Content-Length", strconv.FormatInt(contentlen, 10))
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

func (r *GoResponse) Destroy() {
	if c, ok := r.Writer.(io.Closer); ok {
		c.Close()
	}
	r.Goheader.Source = nil
	r.Original = nil
	r.Writer = nil
}

func (r *GoResponse) SetResponse(w http.ResponseWriter) {
	r.Original = w
	r.Writer = w
	r.Goheader.Source = r
	r.Goheader.isResponse = true

}
func (r *GoHeader) SetCookie(cookie string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Add("Set-Cookie", cookie)
	}
}
func (r *GoHeader) GetCookie(key string) (value ServerCookie, err error) {
	if !r.isResponse {
		var cookie *http.Cookie
		if cookie, err = r.Source.(*GoRequest).Original.Cookie(key); err == nil {
			value = GoCookie(*cookie)

		}

	}
	return
}
func (r *GoHeader) Set(key string, value string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Set(key, value)
	}
}
func (r *GoHeader) Add(key string, value string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Add(key, value)
	}
}
func (r *GoHeader) Del(key string) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.Header().Del(key)
	}
}
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
func (r *GoHeader) SetStatus(statusCode int) {
	if r.isResponse {
		r.Source.(*GoResponse).Original.WriteHeader(statusCode)
	}
}
func (r GoCookie) GetValue() string {
	return r.Value
}
func (f *GoMultipartForm) GetFiles() map[string][]*multipart.FileHeader {
	return f.Form.File
}
func (f *GoMultipartForm) GetValues() url.Values {
	return url.Values(f.Form.Value)
}
func (f *GoMultipartForm) RemoveAll() error {
	return f.Form.RemoveAll()
}
func (g *GoWebSocket) MessageSendJSON(v interface{}) error {
	return websocket.JSON.Send(g.Conn, v)
}
func (g *GoWebSocket) MessageReceiveJSON(v interface{}) error {
	return websocket.Message.Receive(g.Conn, v)
}
