package controllers

import (
	"github.com/robfig/revel"
	"os"
	fpath "path/filepath"
	"strings"
)

type Static struct {
	*revel.Controller
}

func (c Static) ServeDir(prefix, filepath string) revel.Result {
	var basePath, moduleName string

	// Handle module paths
	if i := strings.Index(prefix, "MODULE:"); i != -1 {
		// strip the leading "MODULE:"
		prefix = prefix[7:]
		if i := strings.Index(prefix, ":"); i != -1 {
			moduleName, prefix = prefix[:i], prefix[i+1:]
		}

		// Find the module path
		for _, module := range revel.Modules {
			if module.Name == moduleName {
				basePath = module.Path
				break
			}
		}
	} else {
		if !fpath.IsAbs(prefix) {
			basePath = revel.BasePath
		}
	}

	fname := fpath.Join(basePath, fpath.FromSlash(prefix), fpath.FromSlash(filepath))
	file, err := os.Open(fname)
	if os.IsNotExist(err) {
		revel.WARN.Printf("File not found (%s): %s ", fname, err)
		return c.NotFound("")
	} else if err != nil {
		revel.WARN.Printf("Problem opening file (%s): %s ", fname, err)
		return c.NotFound("This was found but not sure why we couldn't open it.")
	}
	return c.RenderFile(file, "")
}

func (c Static) ServeFile(filepath string) revel.Result {
	if !fpath.IsAbs(filepath) {
		return c.ServeDir("", filepath)
	}
	return c.ServeDir("/", filepath)
}
