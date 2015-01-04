package controllers

import (
	"github.com/revel/revel"
	"os"
	fpath "path/filepath"
	"strings"
	"syscall"
)

type Static struct {
	*revel.Controller
}

// This method handles requests for files. The supplied prefix may be absolute
// or relative. If the prefix is relative it is assumed to be relative to the
// application directory. The filepath may either be just a file or an
// additional filepath to search for the given file. This response may return
// the following responses in the event of an error or invalid request;
//   403(Forbidden): If the prefix filepath combination results in a directory.
//   404(Not found): If the prefix and filepath combination results in a non-existent file.
//   500(Internal Server Error): There are a few edge cases that would likely indicate some configuration error outside of revel.
//
// Note that when defining routes in routes/conf the parameters must not have
// spaces around the comma.
//   Bad:  Static.Serve("public/img", "favicon.png")
//   Good: Static.Serve("public/img","favicon.png")
//
// Examples:
// Serving a directory
//   Route (conf/routes):
//     GET /public/{<.*>filepath} Static.Serve("public")
//   Request:
//     public/js/sessvars.js
//   Calls
//     Static.Serve("public","js/sessvars.js")
//
// Serving a file
//   Route (conf/routes):
//     GET /favicon.ico Static.Serve("public/img","favicon.png")
//   Request:
//     favicon.ico
//   Calls:
//     Static.Serve("public/img", "favicon.png")
func (c Static) Serve(prefix, filepath string) revel.Result {
	// Fix for #503.
	prefix = c.Params.Fixed.Get("prefix")
	if prefix == "" {
		return c.NotFound("")
	}

	return serve(c, prefix, filepath)
}

// This method allows modules to serve binary files. The parameters are the same
// as Static.Serve with the additional module name pre-pended to the list of
// arguments.
func (c Static) ServeModule(moduleName, prefix, filepath string) revel.Result {
	// Fix for #503.
	prefix = c.Params.Fixed.Get("prefix")
	if prefix == "" {
		return c.NotFound("")
	}

	var basePath string
	for _, module := range revel.Modules {
		if module.Name == moduleName {
			basePath = module.Path
		}
	}

	absPath := fpath.Join(basePath, fpath.FromSlash(prefix))

	return serve(c, absPath, filepath)
}


// This method allows static serving of application files in a verified manner.
func serve(c Static, prefix, filepath string) revel.Result {
	var basePath string
	if !fpath.IsAbs(prefix) {
		basePath = revel.BasePath
	}

	basePathPrefix := fpath.Join(basePath, fpath.FromSlash(prefix))
	fname := fpath.Join(basePathPrefix, fpath.FromSlash(filepath))
	// Verify the request file path is within the application's scope of access
	if !strings.HasPrefix(fname, basePathPrefix) {
		revel.WARN.Printf("Attempted to read file outside of base path: %s", fname)
		return c.NotFound("")
	}

	// Verify file path is accessible
	finfo, err := os.Stat(fname)
	if err != nil {
		if os.IsNotExist(err) || err.(*os.PathError).Err == syscall.ENOTDIR {
			revel.WARN.Printf("File not found (%s): %s ", fname, err)
			return c.NotFound("File not found")
		}
		revel.ERROR.Printf("Error trying to get fileinfo for '%s': %s", fname, err)
		return c.RenderError(err)
	}

	// Disallow directory listing
	if finfo.Mode().IsDir() {
		revel.WARN.Printf("Attempted directory listing of %s", fname)
		return c.Forbidden("Directory listing not allowed")
	}

	// Open request file path
	file, err := os.Open(fname)
	if err != nil {
		if os.IsNotExist(err) {
			revel.WARN.Printf("File not found (%s): %s ", fname, err)
			return c.NotFound("File not found")
		}
		revel.ERROR.Printf("Error opening '%s': %s", fname, err)
		return c.RenderError(err)
	}
	return c.RenderFile(file, revel.Inline)
}
