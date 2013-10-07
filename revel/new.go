package main

import (
	"bytes"
	"fmt"
	"go/build"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"

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

Skeleton is an optional argument, and should be relative to $GOROOT.

For example:

    revel new import/path/helloworld

    revel new import/path/helloworld import/path/skeleton
`,
}

func init() {
	rand.Seed(time.Now().Unix())
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

	// specifying skeleton
	if len(args) == 2 { // user specified
		sname := args[1]
		_, errS := build.Import(sname, "", build.FindOnly)
		if errS != nil {
			fmt.Println("go get'n: ", args[1])

			// Execute "go get <pkg>"
			getCmd := exec.Command(goExec, "get", "-d", sname)
			glog.V(1).Infoln("Exec:", getCmd.Args)
			getOutput, errG := getCmd.CombinedOutput()

			// check getOutput for no buildible string
			bpos := bytes.Index(getOutput, []byte("no buildable Go source files in"))
			if errG != nil && bpos == -1 {
				fmt.Fprintf(os.Stderr, "Abort: Could not find or 'go get' Skeleton  source code: %s\n%s\n", getOutput, sname)
				return
			}
		}
		skeletonBase = filepath.Join(srcRoot, sname)

	} else { // use the revel default (bootstrap)
		skeletonBase = filepath.Join(revelPkg.Dir, "skeleton")
	}

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
