package revel

import (
	"net/http"
)

func NewTestController(w http.ResponseWriter, r *http.Request) *Controller{
	context := NewGOContext(nil)
	context.Request.SetRequest(r)
	context.Response.SetResponse(w)
	c := NewController(context)
	return c
}

