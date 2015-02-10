package revel

import (
	"github.com/shaoshing/train"
	"strings"
)

// Server /assets with [train]
// https://github.com/shaoshing/train
var AssetsFilter = func(c *Controller, fc []Filter) {
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/assets") {
		train.ServeRequest(c.Response.Out, c.Request.Request)
	} else {
		fc[0](c, fc[1:])
	}
}

func init() {
	train.ConfigureHttpHandler(nil)
	train.Config.SASS.DebugInfo = DevMode
	train.Config.Verbose = DevMode
	train.Config.BundleAssets = !DevMode
}
