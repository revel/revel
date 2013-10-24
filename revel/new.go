package main

import (
	"bytes"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/robfig/revel"
)

var cmdNew = &Command{
	UsageLine: "new [path] [skeleton]",
	Short:     "create a skeleton Revel application",
	Long: `
New creates a few files to get a new Revel application running quickly.

It puts all of the files in the given import path, taking the final element in
the path to be the app name.

Skeleton is an optional argument, provided as an import path

For example:

    revel new import/path/helloworld

    revel new import/path/helloworld import/path/skeleton
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

	goExec, errGE := exec.LookPath("go")
	if errGE != nil {
		glog.Fatalf("Go executable not found in PATH.")
	}

	importPath := args[0]
	if filepath.IsAbs(importPath) {
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

	srcRoot := filepath.Join(filepath.SplitList(gopath)[0], "src")
	appDir := filepath.Join(srcRoot, filepath.FromSlash(importPath))
	err = os.MkdirAll(appDir, 0777)
	panicOnError(err, "Failed to create directory "+appDir)

	// checking and setting skeleton
	skeletonBase = getSkeletonBase(args)

	mustCopyDir(appDir, skeletonBase, map[string]interface{}{
		// app.conf
		"AppName": filepath.Base(appDir),
		"Secret":  genSecret(),
	})

	// Dotfiles are skipped by mustCopyDir, so we have to explicitly copy the .gitignore.
	gitignore := ".gitignore"
	mustCopyFile(filepath.Join(appDir, gitignore), filepath.Join(skeletonBase, gitignore))

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

// getSkeletonBase determines the correct skeleton
// base from the command line arguments.
// If none is specified, then the revel default is used.
// in:   command line arguments to revel
// out:  the source directory for the skeleton
func getSkeletonBase(args []string) string {
	if len(args) == 2 { // user specified
		skeleton_name := args[1]
		_, errS := build.Import(skeleton_name, "", build.FindOnly)
		if errS != nil {
			// Execute "go get <pkg>"
			getCmd := exec.Command(goExec, "get", "-d", skeleton_name)
			glog.V(1).Infoln("Exec:", getCmd.Args)
			getOutput, errG := getCmd.CombinedOutput()

			// check getOutput for no buildible string
			bpos := bytes.Index(getOutput, []byte("no buildable Go source files in"))
			if errG != nil && bpos == -1 {
				fmt.Fprintf(os.Stderr, "Abort: Could not find or 'go get' Skeleton  source code: %s\n%s\n", getOutput, skeleton_name)
				return
			}
		}
		return filepath.Join(srcRoot, skeleton_name)

	} else { // use the revel default (bootstrap)
		return filepath.Join(revelPkg.Dir, "skeleton")
	}
}
