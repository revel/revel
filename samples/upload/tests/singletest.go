package tests

import (
	"net/url"

	"github.com/revel/revel"
)

type SingleTest struct {
	revel.TestSuite
}

func (t *SingleTest) TestThatSingleAvatarUploadWorks() {
	// Make sure file is required.
	t.PostFile("/single/HandleUpload", url.Values{}, url.Values{
		"avatar": {},
	})
	t.AssertOk()
	t.AssertContains("Upload demo")
	t.AssertContains("Required")

	// Make sure incorrect format is not being accepted.
	t.PostFile("/single/HandleUpload", url.Values{}, url.Values{
		"avatar": {"github.com/revel/revel/samples/upload/public/css/bootstrap.css"},
	})
	t.AssertOk()
	t.AssertContains("Incorrect file format")

	// Ensure low resolution avatar cannot be uploaded.
	t.PostFile("/single/HandleUpload", url.Values{}, url.Values{
		"avatar": {"github.com/revel/revel/samples/upload/public/img/favicon.png"},
	})
	t.AssertOk()
	t.AssertContains("Minimum allowed resolution is 150x150px")

	// Check whether correct avatar is uploaded.
	t.PostFile("/single/HandleUpload", url.Values{}, url.Values{
		"avatar": {"github.com/revel/revel/samples/upload/public/img/glyphicons-halflings.png"},
	})
	t.AssertOk()
	t.AssertContains("image/png")
	t.AssertContains("glyphicons-halflings.png")
	t.AssertContains("Successfully uploaded")
}
