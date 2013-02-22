package controllers

import (
	"github.com/robfig/revel"
	"os"
	fpath "path/filepath"
)

type Static struct {
	*revel.Controller
}

func (c Static) ServeDir(prefix, filepath string) revel.Result {
	var basePath string

	if !fpath.IsAbs(prefix) {
		basePath = revel.BasePath
	}

	fname := fpath.Join(basePath, fpath.FromSlash(prefix), fpath.FromSlash(filepath))

	finfo, err := os.Stat(fname)

	if err == nil {
		if finfo.Mode().IsDir() {
			revel.WARN.Printf("Attempted directory listing of %s", fname)
			return c.Forbidden("Directory listing not allowed")
		}
		file, err := os.Open(fname)
		if os.IsNotExist(err) {
			revel.WARN.Printf("File not found (%s): %s ", fname, err)
			return c.NotFound("File not found")
		} else if err != nil {
			revel.WARN.Printf("Problem opening file (%s): %s ", fname, err)
			return c.RenderError(err)
		}
		return c.RenderFile(file, "")
	} else {
		revel.ERROR.Printf("Error trying to get fileinfo for '%s': %s", fname, err)
	}

	return c.RenderError(err)

}

func (c Static) ServeFile(filepath string) revel.Result {
	if !fpath.IsAbs(filepath) {
		return c.ServeDir("", filepath)
	}
	return c.ServeDir("/", filepath)
}

func (c Static) ServeModuleDir(moduleName, prefix, filepath string) revel.Result {
	var basePath string
	for _, module := range revel.Modules {
		if module.Name == moduleName {
			basePath = module.Path
		}
	}

	absPath := fpath.Join(basePath, fpath.FromSlash(prefix), fpath.FromSlash(filepath))

	return c.ServeDir("/", absPath)
}
