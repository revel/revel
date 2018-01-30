// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"fmt"
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
	if error != nil {
		utilLog.Error("PanicFilter: Caught panic", "error", err, "stack", error.Stack)
		if DevMode {
			fmt.Println(err)
			fmt.Println(error.Stack)
		}
	} else {
		utilLog.Error("PanicFilter: Caught panic, unable to determine stack location", "error", err, "stack", string(debug.Stack()))
		if DevMode {
			fmt.Println(err)
			fmt.Println("stack", string(debug.Stack()))
		}
	}

	if error == nil && DevMode {
		// Only show the sensitive information in the debug stack trace in development mode, not production
		c.Response.SetStatus(http.StatusInternalServerError)
		_, _ = c.Response.GetWriter().Write(debug.Stack())
		return
	}

	c.Result = c.RenderError(error)
}
