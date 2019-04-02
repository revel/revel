// Copyright (c) 2019 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"testing"
)

type bafTestController struct {
	*Controller
}

func (c bafTestController) Before() (Result, bafTestController) {
	return c.Redirect("http://www.example.com"), c
}

func (c bafTestController) Index() Result {
	// We shouldn't get here
	panic("Should not be called")
}

type failingFilter struct {
	t *testing.T
}

func (f failingFilter) FailIfCalled(c *Controller, filterChain []Filter) {
	f.t.Error("Filter should not have been called")
}

func TestInterceptorsNotCalledIfBeforeReturns(t *testing.T) {
	Init("prod", "github.com/revel/revel/testdata", "")
	controllers = make(map[string]*ControllerType)
	RegisterController((*bafTestController)(nil), []*MethodType{
		{
			Name: "Before",
		},
		{
			Name: "Index",
		},
	})

	c := NewControllerEmpty()
	err := c.SetAction("bafTestController", "Index")
	if err != nil {
		t.Error(err.Error())
	}

	BeforeAfterFilter(c, []Filter{failingFilter{t}.FailIfCalled})
}
