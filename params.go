package revel

import (
	"mime/multipart"
	"net/url"
	"os"
	"reflect"
)

// These provide a unified view of the request params.
// Includes:
// - URL query string
// - Form values
// - File uploads
type Params struct {
	url.Values
	Files    map[string][]*multipart.FileHeader
	tmpFiles []*os.File // Temp files used during the request.
}

func ParseParams(req *Request) *Params {
	var files map[string][]*multipart.FileHeader

	// Always want the url parameters.
	values := req.URL.Query()

	// Parse the body depending on the content type.
	switch req.ContentType {
	case "application/x-www-form-urlencoded":
		// Typical form.
		if err := req.ParseForm(); err != nil {
			WARN.Println("Error parsing request body:", err)
		} else {
			for key, vals := range req.Form {
				for _, val := range vals {
					values.Add(key, val)
				}
			}
		}

	case "multipart/form-data":
		// Multipart form.
		// TODO: Extract the multipart form param so app can set it.
		if err := req.ParseMultipartForm(32 << 20 /* 32 MB */); err != nil {
			WARN.Println("Error parsing request body:", err)
		} else {
			for key, vals := range req.MultipartForm.Value {
				for _, val := range vals {
					values.Add(key, val)
				}
			}
			files = req.MultipartForm.File
		}
	}

	return &Params{Values: values, Files: files}
}

func (p *Params) Bind(name string, typ reflect.Type) reflect.Value {
	return Bind(p, name, typ)
}

type ParamsFilter struct{}

func (f ParamsFilter) Call(c *Controller, fc FilterChain) {
	c.Params = ParseParams(c.Request)

	// Clean up from the request.
	defer func() {
		// Delete temp files.
		if c.Request.MultipartForm != nil {
			err := c.Request.MultipartForm.RemoveAll()
			if err != nil {
				WARN.Println("Error removing temporary files:", err)
			}
		}

		for _, tmpFile := range c.Params.tmpFiles {
			err := os.Remove(tmpFile.Name())
			if err != nil {
				WARN.Println("Could not remove upload temp file:", err)
			}
		}
	}()

	fc.Call(c)
}
