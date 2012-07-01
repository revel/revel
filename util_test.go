package rev

import "testing"

func TestContentTypeByFilename(t *testing.T) {
	testCases := map[string]string{
		"xyz.jpg":       "image/jpeg",
		"helloworld.c":  "text/x-c; charset=utf-8",
		"helloworld.":   "application/octet-stream",
		"helloworld":    "application/octet-stream",
		"hello.world.c": "text/x-c; charset=utf-8",
	}
	loadMimeConfig()
	for filename, expected := range testCases {
		actual := ContentTypeByFilename(filename)
		if actual != expected {
			t.Errorf("%s: %s, Expected %s", filename, actual, expected)
		}
	}
}
