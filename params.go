// Copyright (c) 2012-2017 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"mime/multipart"
	"net/url"
	"os"
	"reflect"
)

// Params provides a unified view of the request params.
// Includes:
// - URL query string
// - Form values
// - File uploads
//
// Warning: param maps other than Values may be nil if there were none.
type Params struct {
	url.Values // A unified view of all the individual param maps below.

	// Set by the router
	Fixed url.Values // Fixed parameters from the route, e.g. App.Action("fixed param")
	Route url.Values // Parameters extracted from the route,  e.g. /customers/{id}

	// Set by the ParamsFilter
	Query url.Values // Parameters from the query string, e.g. /index?limit=10
	Form  url.Values // Parameters from the request body.

	Files    map[string][]*multipart.FileHeader // Files uploaded in a multipart form
	tmpFiles []*os.File                         // Temp files used during the request.
}

// ParseParams parses the `http.Request` params into `revel.Controller.Params`
func ParseParams(params *Params, req *Request) {
	params.Query = req.URL.Query()

	// Parse the body depending on the content type.
	switch req.ContentType {
	case "application/x-www-form-urlencoded":
		// Typical form.
		if err := req.ParseForm(); err != nil {
			WARN.Println("Error parsing request body:", err)
		} else {
			params.Form = req.Form
		}

	case "multipart/form-data":
		// Multipart form.
		// TODO: Extract the multipart form param so app can set it.
		if err := req.ParseMultipartForm(32 << 20 /* 32 MB */); err != nil {
			WARN.Println("Error parsing request body:", err)
		} else {
			params.Form = req.MultipartForm.Value
			params.Files = req.MultipartForm.File
		}
	}

	params.Values = params.calcValues()
}

// Bind looks for the named parameter, converts it to the requested type, and
// writes it into "dest", which must be settable.  If the value can not be
// parsed, "dest" is set to the zero value.
func (p *Params) Bind(dest interface{}, name string) {
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Ptr {
		panic("revel/params: non-pointer passed to Bind: " + name)
	}
	value = value.Elem()
	if !value.CanSet() {
		panic("revel/params: non-settable variable passed to Bind: " + name)
	}
	value.Set(Bind(p, name, value.Type()))
}

// calcValues returns a unified view of the component param maps.
func (p *Params) calcValues() url.Values {
	numParams := len(p.Query) + len(p.Fixed) + len(p.Route) + len(p.Form)

	// If there were no params, return an empty map.
	if numParams == 0 {
		return make(url.Values, 0)
	}

	// If only one of the param sources has anything, return that directly.
	switch numParams {
	case len(p.Query):
		return p.Query
	case len(p.Route):
		return p.Route
	case len(p.Fixed):
		return p.Fixed
	case len(p.Form):
		return p.Form
	}

	// Copy everything into a param map,
	// order of priority is least to most trusted
	values := make(url.Values, numParams)

	// ?query vars first
	for k, v := range p.Query {
		values[k] = append(values[k], v...)
	}
	// form vars overwrite
	for k, v := range p.Form {
		values[k] = append(values[k], v...)
	}
	// :/path vars overwrite
	for k, v := range p.Route {
		values[k] = append(values[k], v...)
	}
	// fixed vars overwrite
	for k, v := range p.Fixed {
		values[k] = append(values[k], v...)
	}

	return values
}

func ParamsFilter(c *Controller, fc []Filter) {
	ParseParams(c.Params, c.Request)

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

	fc[0](c, fc[1:])
}
