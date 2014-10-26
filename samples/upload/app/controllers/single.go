package controllers

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/revel/revel/samples/upload/app/routes"

	"github.com/revel/revel"
)

const (
	_      = iota
	KB int = 1 << (10 * iota)
	MB
	GB
)

type Single struct {
	App
}

func (c *Single) Upload() revel.Result {
	return c.Render()
}

func (c *Single) HandleUpload(avatar []byte) revel.Result {
	// Validation rules.
	c.Validation.Required(avatar)
	c.Validation.MinSize(avatar, 2*KB).
		Message("Minimum a file size of 2KB expected")
	c.Validation.MaxSize(avatar, 2*MB).
		Message("File cannot be larger than 2MB")

	// Check format of the file.
	conf, format, err := image.DecodeConfig(bytes.NewReader(avatar))
	c.Validation.Required(err == nil).Key("avatar").
		Message("Incorrect file format")
	c.Validation.Required(format == "jpeg" || format == "png").Key("avatar").
		Message("JPEG or PNG file format is expected")

	// Check resolution.
	c.Validation.Required(conf.Height >= 150 && conf.Width >= 150).Key("avatar").
		Message("Minimum allowed resolution is 150x150px")

	// Handle errors.
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(routes.Single.Upload())
	}

	return c.RenderJson(FileInfo{
		ContentType: c.Params.Files["avatar"][0].Header.Get("Content-Type"),
		Filename:    c.Params.Files["avatar"][0].Filename,
		RealFormat:  format,
		Resolution:  fmt.Sprintf("%dx%d", conf.Width, conf.Height),
		Size:        len(avatar),
		Status:      "Successfully uploaded",
	})
}
