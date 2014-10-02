package controllers

import (
	"github.com/revel/revel/samples/upload/app/routes"

	"github.com/revel/revel"
)

type Multiple struct {
	App
}

func (c *Multiple) Upload() revel.Result {
	return c.Render()
}

func (c *Multiple) HandleUpload() revel.Result {
	var files [][]byte
	c.Params.Bind(&files, "file")

	// Make sure at least 2 but no more than 3 files are submitted.
	c.Validation.MinSize(files, 2).Message("You cannot submit less than 2 files")
	c.Validation.MaxSize(files, 3).Message("You cannot submit more than 3 files")

	// Handle errors.
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(routes.Multiple.Upload())
	}

	// Prepare result.
	filesInfo := make([]FileInfo, len(files))
	for i, _ := range files {
		filesInfo[i] = FileInfo{
			ContentType: c.Params.Files["file[]"][i].Header.Get("Content-Type"),
			Filename:    c.Params.Files["file[]"][i].Filename,
			Size:        len(files[i]),
		}
	}

	return c.RenderJson(map[string]interface{}{
		"Count":  len(files),
		"Files":  filesInfo,
		"Status": "Successfully uploaded",
	})
}
