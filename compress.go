// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zlib"
)

var compressionTypes = [...]string{
	"gzip",
	"deflate",
}

var compressableMimes = [...]string{
	"text/plain",
	"text/html",
	"text/xml",
	"text/css",
	"application/json",
	"application/xml",
	"application/xhtml+xml",
	"application/rss+xml",
	"application/javascript",
	"application/x-javascript",
}

// Local log instance for this class
var compressLog = RevelLog.New("section", "compress")

// WriteFlusher interface for compress writer
type WriteFlusher interface {
	io.Writer
	io.Closer
	Flush() error
}

type CompressResponseWriter struct {
	Header             *BufferedServerHeader
	ControllerResponse *Response
	OriginalWriter     io.Writer
	compressWriter     WriteFlusher
	compressionType    string
	headersWritten     bool
	closeNotify        chan bool
	parentNotify       <-chan bool
	closed             bool
}

// CompressFilter does compression of response body in gzip/deflate if
// `results.compressed=true` in the app.conf
func CompressFilter(c *Controller, fc []Filter) {
	if c.Response.Out.internalHeader.Server != nil && Config.BoolDefault("results.compressed", false) {
		if c.Response.Status != http.StatusNoContent && c.Response.Status != http.StatusNotModified {
			if found, compressType, compressWriter := detectCompressionType(c.Request, c.Response); found {
				writer := CompressResponseWriter{
					ControllerResponse: c.Response,
					OriginalWriter:     c.Response.GetWriter(),
					compressWriter:     compressWriter,
					compressionType:    compressType,
					headersWritten:     false,
					closeNotify:        make(chan bool, 1),
					closed:             false,
				}
				// Swap out the header with our own
				writer.Header = NewBufferedServerHeader(c.Response.Out.internalHeader.Server)
				c.Response.Out.internalHeader.Server = writer.Header
				if w, ok := c.Response.GetWriter().(http.CloseNotifier); ok {
					writer.parentNotify = w.CloseNotify()
				}
				c.Response.SetWriter(&writer)
			}
		} else {
			compressLog.Debug("CompressFilter: Compression disabled for response ", "status", c.Response.Status)
		}
	}
	fc[0](c, fc[1:])
}

func (c CompressResponseWriter) CloseNotify() <-chan bool {
	if c.parentNotify != nil {
		return c.parentNotify
	}
	return c.closeNotify
}

func (c *CompressResponseWriter) prepareHeaders() {
	if c.compressionType != "" {
		responseMime := ""
		if t := c.Header.Get("Content-Type"); len(t) > 0 {
			responseMime = t[0]
		}
		responseMime = strings.TrimSpace(strings.SplitN(responseMime, ";", 2)[0])
		shouldEncode := false

		if len(c.Header.Get("Content-Encoding")) == 0 {
			for _, compressableMime := range compressableMimes {
				if responseMime == compressableMime {
					shouldEncode = true
					c.Header.Set("Content-Encoding", c.compressionType)
					c.Header.Del("Content-Length")
					break
				}
			}
		}

		if !shouldEncode {
			c.compressWriter = nil
			c.compressionType = ""
		}
	}
	c.Header.Release()
}

func (c *CompressResponseWriter) WriteHeader(status int) {
	c.headersWritten = true
	c.prepareHeaders()
	c.Header.SetStatus(status)
}

func (c *CompressResponseWriter) Close() error {
	if !c.headersWritten {
		c.prepareHeaders()
	}
	if c.compressionType != "" {
		c.Header.Del("Content-Length")
		if err := c.compressWriter.Close(); err != nil {
			// TODO When writing directly to stream, an error will be generated
			compressLog.Error("Close: Error closing compress writer", "type", c.compressionType, "error", err)
		}

	}
	// Non-blocking write to the closenotifier, if we for some reason should
	// get called multiple times
	select {
	case c.closeNotify <- true:
	default:
	}
	c.closed = true
	return nil
}

func (c *CompressResponseWriter) Write(b []byte) (int, error) {
	// Abort if parent has been closed
	if c.parentNotify != nil {
		select {
		case <-c.parentNotify:
			return 0, io.ErrClosedPipe
		default:
		}
	}
	// Abort if we ourselves have been closed
	if c.closed {
		return 0, io.ErrClosedPipe
	}

	if !c.headersWritten {
		c.prepareHeaders()
		c.headersWritten = true
	}
	if c.compressionType != "" {
		return c.compressWriter.Write(b)
	}
	return c.OriginalWriter.Write(b)
}

// DetectCompressionType method detects the compression type
// from header "Accept-Encoding"
func detectCompressionType(req *Request, resp *Response) (found bool, compressionType string, compressionKind WriteFlusher) {
	if Config.BoolDefault("results.compressed", false) {
		acceptedEncodings := strings.Split(req.GetHttpHeader("Accept-Encoding"), ",")

		largestQ := 0.0
		chosenEncoding := len(compressionTypes)

		// I have fixed one edge case for issue #914
		// But it's better to cover all possible edge cases or
		// Adapt to https://github.com/golang/gddo/blob/master/httputil/header/header.go#L172
		for _, encoding := range acceptedEncodings {
			encoding = strings.TrimSpace(encoding)
			encodingParts := strings.SplitN(encoding, ";", 2)

			// If we are the format "gzip;q=0.8"
			if len(encodingParts) > 1 {
				q := strings.TrimSpace(encodingParts[1])
				if len(q) == 0 || !strings.HasPrefix(q, "q=") {
					continue
				}

				// Strip off the q=
				num, err := strconv.ParseFloat(q[2:], 32)
				if err != nil {
					continue
				}

				if num >= largestQ && num > 0 {
					if encodingParts[0] == "*" {
						chosenEncoding = 0
						largestQ = num
						continue
					}
					for i, encoding := range compressionTypes {
						if encoding == encodingParts[0] {
							if i < chosenEncoding {
								largestQ = num
								chosenEncoding = i
							}
							break
						}
					}
				}
			} else {
				// If we can accept anything, chose our preferred method.
				if encodingParts[0] == "*" {
					chosenEncoding = 0
					largestQ = 1
					break
				}
				// This is for just plain "gzip"
				for i, encoding := range compressionTypes {
					if encoding == encodingParts[0] {
						if i < chosenEncoding {
							largestQ = 1.0
							chosenEncoding = i
						}
						break
					}
				}
			}
		}

		if largestQ == 0 {
			return
		}

		compressionType = compressionTypes[chosenEncoding]

		switch compressionType {
		case "gzip":
			compressionKind = gzip.NewWriter(resp.GetWriter())
			found = true
		case "deflate":
			compressionKind = zlib.NewWriter(resp.GetWriter())
			found = true
		}
	}
	return
}

// BufferedServerHeader will not send content out until the Released is called, from that point on it will act normally
// It implements all the ServerHeader
type BufferedServerHeader struct {
	cookieList []string
	headerMap  map[string][]string
	status     int
	released   bool
	original   ServerHeader
}

func NewBufferedServerHeader(o ServerHeader) *BufferedServerHeader {
	return &BufferedServerHeader{original: o, headerMap: map[string][]string{}}
}
func (bsh *BufferedServerHeader) SetCookie(cookie string) {
	if bsh.released {
		bsh.original.SetCookie(cookie)
	} else {
		bsh.cookieList = append(bsh.cookieList, cookie)
	}
}
func (bsh *BufferedServerHeader) GetCookie(key string) (value ServerCookie, err error) {
	return bsh.original.GetCookie(key)
}
func (bsh *BufferedServerHeader) Set(key string, value string) {
	if bsh.released {
		bsh.original.Set(key, value)
	} else {
		bsh.headerMap[key] = []string{value}
	}
}
func (bsh *BufferedServerHeader) Add(key string, value string) {
	if bsh.released {
		bsh.original.Set(key, value)
	} else {
		old := []string{}
		if v, found := bsh.headerMap[key]; found {
			old = v
		}
		bsh.headerMap[key] = append(old, value)
	}

}
func (bsh *BufferedServerHeader) Del(key string) {
	if bsh.released {
		bsh.original.Del(key)
	} else {
		delete(bsh.headerMap, key)
	}

}
func (bsh *BufferedServerHeader) Get(key string) (value []string) {
	if bsh.released {
		value = bsh.original.Get(key)
	} else {
		if v, found := bsh.headerMap[key]; found && len(v) > 0 {
			value = v
		} else {
			value = bsh.original.Get(key)
		}
	}
	return
}
func (bsh *BufferedServerHeader) SetStatus(statusCode int) {
	if bsh.released {
		bsh.original.SetStatus(statusCode)
	} else {
		bsh.status = statusCode
	}
}
func (bsh *BufferedServerHeader) Release() {
	bsh.released = true
	bsh.original.SetStatus(bsh.status)
	for k, v := range bsh.headerMap {
		for _, r := range v {
			bsh.original.Set(k, r)
		}
	}
	for _, c := range bsh.cookieList {
		bsh.original.SetCookie(c)
	}
}
