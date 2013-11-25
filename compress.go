package revel

import (
	"compress/gzip"
	"compress/zlib"
	"net/http"
	"strings"
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
	"application/xml",
	"application/xhtml+xml",
	"application/rss+xml",
	"application/javascript",
	"application/x-javascript",
}

type WriteFlusher interface {
	Write([]byte) (int, error)
	Flush() error
}

type CompressResponseWriter struct {
	http.ResponseWriter
	compressWriter  WriteFlusher
	compressionType string
	headersWritten  bool
}

func CompressFilter(c *Controller, fc []Filter) {
	writer := CompressResponseWriter{c.Response.Out, nil, "", false}
	writer.DetectCompressionType(c.Request, c.Response)
	c.Response.Out = &writer

	fc[0](c, fc[1:])
}

func (c *CompressResponseWriter) prepareHeaders() {
	mime := c.Header().Get("Content-Type")
	shouldEncode := false

	for _, mimeType := range compressableMimes {
		if strings.HasPrefix(mime, mimeType) {
			shouldEncode = true
			c.Header().Set("Content-Encoding", c.compressionType)
			c.Header().Del("Content-Length")
			break
		}
	}

	if !shouldEncode {
		c.compressWriter = nil
		c.compressionType = ""
	}
}

func (c *CompressResponseWriter) WriteHeader(status int) {
	c.headersWritten = true
	c.prepareHeaders()
	c.ResponseWriter.WriteHeader(status)
}

func (c *CompressResponseWriter) Write(b []byte) (int, error) {
	if !c.headersWritten {
		c.prepareHeaders()
		c.headersWritten = true
	}

	if c.compressionType != "" {
		defer c.compressWriter.Flush()
		return c.compressWriter.Write(b)
	} else {
		return c.ResponseWriter.Write(b)
	}
}

func (c *CompressResponseWriter) DetectCompressionType(req *Request, resp *Response) {
	if Config.BoolDefault("results.compressed", false) {
		acceptedEncodings := strings.Split(req.Request.Header.Get("Accept-Encoding"), ",")

		for i := 0; i < len(acceptedEncodings) && c.compressionType == ""; i++ {
			switch strings.TrimSpace(acceptedEncodings[i]) {
			case "gzip":
				c.compressionType = "gzip"
				c.compressWriter = gzip.NewWriter(resp.Out)
			case "deflate":
				c.compressionType = "deflate"
				c.compressWriter = zlib.NewWriter(resp.Out)
			}
		}
	}
}
