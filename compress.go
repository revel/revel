package revel

import (
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"strconv"
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
	"application/json",
	"application/xml",
	"application/xhtml+xml",
	"application/rss+xml",
	"application/javascript",
	"application/x-javascript",
}

type WriteFlusher interface {
	io.Writer
	io.Closer
	Flush() error
}

type CompressResponseWriter struct {
	http.ResponseWriter
	compressWriter  WriteFlusher
	compressionType string
	headersWritten  bool
	closeNotify     chan bool
	parentNotify    <-chan bool
	closed          bool
}

func CompressFilter(c *Controller, fc []Filter) {
	if Config.BoolDefault("results.compressed", false) {
		writer := CompressResponseWriter{c.Response.Out, nil, "", false, make(chan bool, 1), nil, false}
		writer.DetectCompressionType(c.Request, c.Response)
		w, ok := c.Response.Out.(http.CloseNotifier)
		if ok {
			writer.parentNotify = w.CloseNotify()
		}
		c.Response.Out = &writer
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
		responseMime := c.Header().Get("Content-Type")
		responseMime = strings.TrimSpace(strings.SplitN(responseMime, ";", 2)[0])
		shouldEncode := false

		if c.Header().Get("Content-Encoding") == "" {
			for _, compressableMime := range compressableMimes {
				if responseMime == compressableMime {
					shouldEncode = true
					c.Header().Set("Content-Encoding", c.compressionType)
					c.Header().Del("Content-Length")
					break
				}
			}
		}

		if !shouldEncode {
			c.compressWriter = nil
			c.compressionType = ""
		}
	}
}

func (c *CompressResponseWriter) WriteHeader(status int) {
	c.headersWritten = true
	c.prepareHeaders()
	c.ResponseWriter.WriteHeader(status)
}

func (c *CompressResponseWriter) Close() error {
	if c.compressionType != "" {
		c.compressWriter.Close()
	}
	if w, ok := c.ResponseWriter.(io.Closer); ok {
		w.Close()
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
	} else {
		return c.ResponseWriter.Write(b)
	}
}

func (c *CompressResponseWriter) DetectCompressionType(req *Request, resp *Response) {
	if Config.BoolDefault("results.compressed", false) {
		acceptedEncodings := strings.Split(req.Request.Header.Get("Accept-Encoding"), ",")

		largestQ := 0.0
		chosenEncoding := len(compressionTypes)

		for _, encoding := range acceptedEncodings {
			encoding = strings.TrimSpace(encoding)
			encodingParts := strings.SplitN(encoding, ";", 2)

			// If we are the format "gzip;q=0.8"
			if len(encodingParts) > 1 {
				// Strip off the q=
				num, err := strconv.ParseFloat(strings.TrimSpace(encodingParts[1])[2:], 32)
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

		c.compressionType = compressionTypes[chosenEncoding]

		switch c.compressionType {
		case "gzip":
			c.compressWriter = gzip.NewWriter(resp.Out)
		case "deflate":
			c.compressWriter = zlib.NewWriter(resp.Out)
		}
	}
}
