package revel

import (
	"compress/gzip"
	"compress/zlib"
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
	if c.compressionType != "" {
		responseMime := c.Header().Get("Content-Type")
		responseMime = strings.TrimSpace(strings.SplitN(responseMime, ";", 2)[0])
		shouldEncode := false

		for _, compressableMime := range compressableMimes {
			if responseMime == compressableMime {
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
