package controllers

import (
	"github.com/robfig/revel"
	"os"
	"path"
)

type Static struct {
	*revel.Controller
}

func (c Static) ServeDir(prefix, filepath string) revel.Result {
	var basePath, dirName string

	if !path.IsAbs(dirName) {
		basePath = revel.BasePath
	}

	fname := path.Join(basePath, prefix, filepath)
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
	return c.ServeDir("", filepath)
}
