package revel

import (
	"github.com/huacnlee/train"
	"strings"
)

// Server /assets with [train]
// https://github.com/huacnlee/train

var AssetsFilter = func(c *Controller, fc []Filter) {
	if !AssetsCompile {
		fc[0](c, fc[1:])
		return
	}

	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/assets") {
		train.ServeRequest(c.Response.Out, c.Request.Request)
	} else {
		fc[0](c, fc[1:])
	}
}

var (
	initedAssets = false
)

func initAssetsPipeline() {
	if initedAssets {
		return
	}

	train.Config.AssetsPath = AppPath + "/assets"
	train.Config.BundleAssets = true

	if Config.BoolDefault("assets.debug", false) == false {
		train.Config.SASS.DebugInfo = false
		train.Config.Verbose = DevMode
	}

	if AssetsCompile {
		train.ConfigureHttpHandler(nil)
	}

	initedAssets = true
}

func init() {
	TemplateFuncs["javascript_include_tag"] = train.JavascriptTag
	TemplateFuncs["stylesheet_link_tag"] = train.StylesheetTag

	OnAppStart(func() {
		initAssetsPipeline()
	})
}
