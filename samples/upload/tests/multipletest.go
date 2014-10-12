package tests

import (
	"net/url"
	"path"

	"github.com/revel/revel/samples/upload/app/routes"

	"github.com/revel/revel"
)

// MultipleTest is a test suite of Multiple controller.
type MultipleTest struct {
	revel.TestSuite
}

// TestThatMultipleFilesUploadWorks makes sure that Multiple.HandleUpload requires
// multiple files to be uploaded.
func (t *MultipleTest) TestThatMultipleFilesUploadWorks() {
	// Make sure it is not allowed to submit less than 2 files.
	t.PostFile(routes.Multiple.HandleUpload(), url.Values{}, url.Values{
		"file": {
			path.Join(revel.BasePath, "public/img/favicon.png"),
		},
	})
	t.AssertOk()
	t.AssertContains("You cannot submit less than 2 files")

	// Make sure upload of 2 files works.
	t.PostFile(routes.Multiple.HandleUpload(), url.Values{}, url.Values{
		"file[]": {
			path.Join(revel.BasePath, "public/img/favicon.png"),
			path.Join(revel.BasePath, "public/img/glyphicons-halflings.png"),
		},
	})
	t.AssertOk()
	t.AssertContains("Successfully uploaded")
	t.AssertContains("favicon.png")
	t.AssertContains("glyphicons-halflings.png")
	t.AssertContains("image/png")
}
