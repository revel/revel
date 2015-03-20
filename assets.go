package revel

import (
	"github.com/huacnlee/train"
	"strings"
)

// Server /assets with [train]
// https://github.com/shaoshing/train

var AssetsFilter = func(c *Controller, fc []Filter) {
	checkInitAssetsPipeline()
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/assets") {
		train.ServeRequest(c.Response.Out, c.Request.Request)
	} else {
		fc[0](c, fc[1:])
	}
}

var asssetInited bool

func checkInitAssetsPipeline() {
	if asssetInited {
		return
	}

	train.Config.AssetsPath = AppPath + "/assets"
	train.Config.SASS.DebugInfo = false
	train.Config.Verbose = DevMode
	train.Config.BundleAssets = true
	train.ConfigureHttpHandler(nil)

	asssetInited = true
}

func init() {
	TemplateFuncs["javascript_include_tag"] = train.JavascriptTag
	TemplateFuncs["stylesheet_link_tag"] = train.StylesheetTag
}
