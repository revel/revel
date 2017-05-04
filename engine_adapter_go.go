package revel

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
	"io"
	"mime/multipart"
	"net/url"
	"strconv"
)

// Register the GOHttpServer engine

func init() {
	RegisterServerEngine(&GOHttpServer{})
}

type GOHttpServer struct {
	Server               *http.Server
	ServerInit           *EngineInit
	MaxMultipartSize     int64
	goContextStack       *SimpleLockStack
	goMultipartFormStack *SimpleLockStack
}

func (g *GOHttpServer) Init(init *EngineInit) {
	g.MaxMultipartSize = int64(Config.IntDefault("server.request.max.multipart.filesize", 32)) << 20 /* 32 MB */
	g.goContextStack = NewStackLock(Config.IntDefault("server.context.stack", 100),
		Config.IntDefault("server.context.maxstack", 200),
		func() interface{} {
			return NewGOContext(g)
		})
	g.goMultipartFormStack = NewStackLock(Config.IntDefault("server.form.stack", 100),
		Config.IntDefault("server.form.maxstack", 200),
		func() interface{} { return &GOMultipartForm{} })
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
	context := g.goContextStack.Pop().(*GOContext)
	defer func() {
		g.goContextStack.Push(context)
	}()
	context.Request.SetRequest(r)
	context.Response.SetResponse(w)

	if upgrade == "websocket" || upgrade == "Websocket" {
		websocket.Handler(func(ws *websocket.Conn) {
			//Override default Read/Write timeout with sane value for a web socket request
			if err := ws.SetDeadline(time.Now().Add(time.Hour * 24)); err != nil {
				ERROR.Println("SetDeadLine failed:", err)
			}
			r.Method = "WS"
			context.Request.WebSocket = ws
			context.WebSocket = &GOWebsocket{Conn: ws, GOResponse: *context.Response}
			g.ServerInit.Callback(context)
		}).ServeHTTP(w, r)
	} else {
		g.ServerInit.Callback(context)
	}
}

const GO_NATIVE_SERVER_ENGINE = "go"

func (g *GOHttpServer) Name() string {
	return GO_NATIVE_SERVER_ENGINE
}

func (g *GOHttpServer) Stats() map[string]interface{} {
	return map[string]interface{}{
		"Go Engine Context": g.goContextStack.String(),
		"Go Engine Forms":   g.goMultipartFormStack.String(),
	}
}

func (g *GOHttpServer) Engine() interface{} {
	return g.Server
}

func (g *GOHttpServer) Event(event int, args interface{}) {

}

type (
	GOContext struct {
		Request   *GORequest
		Response  *GOResponse
		WebSocket *GOWebsocket
	}
	GORequest struct {
		Original        *http.Request
		FormParsed      bool
		MultiFormParsed bool
		WebSocket       *websocket.Conn
		ParsedForm      *GOMultipartForm
		Goheader        *GOHeader
		Engine          *GOHttpServer
	}

	GOResponse struct {
		Original http.ResponseWriter
		Goheader *GOHeader
		Writer   io.Writer
		Request  *GORequest
		Engine   *GOHttpServer
	}
	GOMultipartForm struct {
		Form *multipart.Form
	}
	GOHeader struct {
		Source     interface{}
		isResponse bool
	}
	GOWebsocket struct {
		Conn *websocket.Conn
		GOResponse
	}
	GoCookie http.Cookie
)

func NewGOContext(instance *GOHttpServer) *GOContext {
	if instance == nil {
		instance = &GOHttpServer{MaxMultipartSize: 32 << 20}
		instance.goContextStack = NewStackLock(100, 200,
			func() interface{} {
				return NewGOContext(instance)
			})
		instance.goMultipartFormStack = NewStackLock(100, 200,
			func() interface{} { return &GOMultipartForm{} })
	}
	c := &GOContext{Request: &GORequest{Goheader: &GOHeader{}, Engine: instance}}
	c.Response = &GOResponse{Goheader: &GOHeader{}, Request: c.Request, Engine: instance}
	return c
}
func (c *GOContext) GetRequest() ServerRequest {
	return c.Request
}
func (c *GOContext) GetResponse() ServerResponse {
	if c.WebSocket != nil {
		return c.WebSocket
	}
	return c.Response
}
func (c *GOContext) Destroy() {
	c.Response.Destroy()
	c.Request.Destroy()
	if c.WebSocket != nil {
		c.WebSocket.Destroy()
	}
}
func (r *GORequest) Get(key int) (value interface{}, err error) {
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
		value = r.Original.URL.RequestURI()
	case HTTP_REMOTE_ADDR:
		value = r.Original.RemoteAddr
	case HTTP_METHOD:
		value = r.Original.Method
	case HTTP_PATH:
		value = r.Original.URL.Path
	case HTTP_HOST:
		value = r.Original.Host
	case HTTP_BODY:
		value = r.Original.Body
	default:
		err = ENGINE_UNKNOWN_GET
	}

	return
}
func (r *GORequest) Set(key int, value interface{}) bool {
	return false
}

func (r *GORequest) GetForm() (url.Values, error) {
	if !r.FormParsed {
		if e := r.Original.ParseForm(); e != nil {
			return nil, e
		}
		r.FormParsed = true
	}
	return r.Original.Form, nil
}
func (r *GORequest) GetMultipartForm() (ServerMultipartForm, error) {
	if !r.MultiFormParsed {
		if e := r.Original.ParseMultipartForm(r.Engine.MaxMultipartSize); e != nil {
			return nil, e
		}
		r.ParsedForm = r.Engine.goMultipartFormStack.Pop().(*GOMultipartForm)
		r.ParsedForm.Form = r.Original.MultipartForm
	}

	return r.ParsedForm, nil
}
func (r *GORequest) GetHeader() ServerHeader {
	return r.Goheader
}
func (r *GORequest) GetRaw() interface{} {
	return r.Original
}
func (r *GORequest) SetRequest(req *http.Request) {
	r.Original = req
	r.Goheader.Source = r
	r.Goheader.isResponse = false

}
func (r *GORequest) Destroy() {
	r.Goheader.Source = nil
	r.Original = nil
	r.FormParsed = false
	r.MultiFormParsed = false
	r.ParsedForm = nil
}
func (r *GOResponse) Get(key int) (value interface{}, err error) {
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
func (r *GOResponse) Set(key int, value interface{}) (set bool) {
	switch key {
	case HTTP_WRITER:
		r.SetWriter(value.(io.Writer))
		set = true
	}
	return
}

func (r *GOResponse) Header() ServerHeader {
	return r.Goheader
}
func (r *GOResponse) GetRaw() interface{} {
	return r.Original
}
func (r *GOResponse) SetWriter(writer io.Writer) {
	r.Writer = writer
}
func (r *GOResponse) WriteStream(name string, contentlen int64, modtime time.Time, reader io.Reader) error {

	// Check to see if the output stream is modified, if not send it using the
	// Native writer
	if _, ok := r.Writer.(http.ResponseWriter); ok {
		if rs, ok := reader.(io.ReadSeeker); ok {
			http.ServeContent(r.Original, r.Request.Original, name, modtime, rs)
		}
	} else {
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

func (r *GOResponse) Destroy() {
	if c, ok := r.Writer.(io.Closer); ok {
		c.Close()
	}
	r.Goheader.Source = nil
	r.Original = nil
	r.Writer = nil
}

func (r *GOResponse) SetResponse(w http.ResponseWriter) {
	r.Original = w
	r.Writer = w
	r.Goheader.Source = r
	r.Goheader.isResponse = true

}
func (r *GOHeader) SetCookie(cookie string) {
	if r.isResponse {
		r.Source.(*GOResponse).Original.Header().Add("Set-Cookie", cookie)
	}
}
func (r *GOHeader) GetCookie(key string) (value ServerCookie, err error) {
	if !r.isResponse {
		var cookie *http.Cookie
		if cookie, err = r.Source.(*GORequest).Original.Cookie(key); err == nil {
			value = GoCookie(*cookie)

		}

	}
	return
}
func (r *GOHeader) Set(key string, value string) {
	if r.isResponse {
		r.Source.(*GOResponse).Original.Header().Set(key, value)
	}
}
func (r *GOHeader) Add(key string, value string) {
	if r.isResponse {
		r.Source.(*GOResponse).Original.Header().Add(key, value)
	}
}
func (r *GOHeader) Del(key string) {
	if r.isResponse {
		r.Source.(*GOResponse).Original.Header().Del(key)
	}
}
func (r *GOHeader) Get(key string) (value string) {
	if !r.isResponse {
		value = r.Source.(*GORequest).Original.Header.Get(key)
	} else {
		value = r.Source.(*GOResponse).Original.Header().Get(key)
	}
	return
}
func (r *GOHeader) SetStatus(statusCode int) {
	if r.isResponse {
		r.Source.(*GOResponse).Original.WriteHeader(statusCode)
	}
}
func (r GoCookie) GetValue() string {
	return r.Value
}
func (f *GOMultipartForm) GetFile() map[string][]*multipart.FileHeader {
	return f.Form.File
}
func (f *GOMultipartForm) GetValue() url.Values {
	return url.Values(f.Form.Value)
}
func (f *GOMultipartForm) RemoveAll() error {
	return f.Form.RemoveAll()
}
func (g *GOWebsocket) MessageSendJson(v interface{}) error {
	return websocket.JSON.Send(g.Conn, v)
}
func (g *GOWebsocket) MessageReceiveJson(v interface{}) error {
	return websocket.Message.Receive(g.Conn, v)
}
