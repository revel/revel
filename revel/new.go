package main

import (
	"fmt"
	"github.com/robfig/revel"
	"go/build"
	"math/rand"
	"os"
	"path"
	"path/filepath"
)

var cmdNew = &Command{
	UsageLine: "new [path]",
	Short:     "create a skeleton Revel application",
	Long: `
New creates a few files to get a new Revel application running quickly.

It puts all of the files in the given import path, taking the final element in
the path to be the app name.

For example:

    revel new import/path/helloworld
`,
}

func init() {
	cmdNew.Run = newApp
}

var (
	appDir       string
	skeletonBase string
)

func newApp(args []string) {
	if len(args) == 0 {
		errorf("No import path given.\nRun 'revel help new' for usage.\n")
	}

	gopath := build.Default.GOPATH
	if gopath == "" {
		errorf("Abort: GOPATH environment variable is not set. " +
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.")
	}

	importPath := args[0]
	if path.IsAbs(importPath) {
		errorf("Abort: '%s' looks like a directory.  Please provide a Go import path instead.",
			importPath)
	}

	_, err := build.Import(importPath, "", build.FindOnly)
	if err == nil {
		fmt.Fprintf(os.Stderr, "Abort: Import path %s already exists.\n", importPath)
		return
	}

	revelPkg, err := build.Import(revel.REVEL_IMPORT_PATH, "", build.FindOnly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Abort: Could not find Revel source code: %s\n", err)
		return
	}

	srcRoot := path.Join(filepath.SplitList(gopath)[0], "src")
	appDir := path.Join(srcRoot, filepath.FromSlash(importPath))
	err = os.MkdirAll(appDir, 0777)
	panicOnError(err, "Failed to create directory "+appDir)

	skeletonBase = path.Join(revelPkg.Dir, "skeleton")
	mustCopyDir(appDir, skeletonBase, map[string]interface{}{
		// app.conf
		"AppName": filepath.Base(appDir),
		"Secret":  genSecret(),
	})

	// Dotfiles are skipped by mustCopyDir, so we have to explicitly copy the .gitignore.
	gitignore := ".gitignore"
	mustCopyFile(path.Join(appDir, gitignore), path.Join(skeletonBase, gitignore))

	fmt.Fprintln(os.Stdout, "Your application is ready:\n  ", appDir)
	fmt.Fprintln(os.Stdout, "\nYou can run it with:\n   revel run", importPath)
}

const alphaNumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func genSecret() string {
	chars := make([]byte, 64)
	for i := 0; i < 64; i++ {
		chars[i] = alphaNumeric[rand.Intn(len(alphaNumeric))]
	}
	return string(chars)
}
