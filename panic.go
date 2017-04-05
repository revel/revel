// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"net/http"
	"runtime/debug"
)

// PanicFilter wraps the action invocation in a protective defer blanket that
// converts panics into 500 error pages.
func PanicFilter(c *Controller, fc []Filter) {
	defer func() {
		if err := recover(); err != nil {
			handleInvocationPanic(c, err)
		}
	}()
	fc[0](c, fc[1:])
}

// This function handles a panic in an action invocation.
// It cleans up the stack trace, logs it, and displays an error page.
func handleInvocationPanic(c *Controller, err interface{}) {
	error := NewErrorFromPanic(err)
	if error == nil && DevMode {
		// Only show the sensitive information in the debug stack trace in development mode, not production
		ERROR.Print(err, "\n", string(debug.Stack()))
		c.Response.Out.Header().SetStatus(http.StatusInternalServerError)
		_, _ = c.Response.Out.GetWriter().Write(debug.Stack())
		return
	}

	ERROR.Print(err, "\n", error.Stack)
	c.Result = c.RenderError(error)
}
