package main

import (
	"fmt"
	"go/build"
	"math/rand"
	"os"
	"path"
	"path/filepath"
)

var cmdNew = &Command{
	UsageLine: "new [path]",
	Short:     "create a skeleton Revel application",
	Long: `~
~ New creates a few files to get a new Revel application running quickly.
~
~ It puts all of the files in the given directory, taking the final element in
~ the path to be the app name.
~
~ For example:
~   rev new github.com/robfig/chatapp`,
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
		errorf("~ No path given.\nRun 'rev help new' for usage.\n")
	}

	_, err := os.Open(args[0])
	if err == nil {
		fmt.Fprintf(os.Stderr, "~ Abort: Directory %s already exists.\n", args[0])
		return
	}

	revelPkg, err := build.Import("github.com/robfig/revel", "", build.FindOnly)
	if err != nil {
		fmt.Fprintln(os.Stderr, "~ Failed to find revel code.")
		return
	}

	err = os.MkdirAll(args[0], 0777)
	panicOnError(err, "Failed to create directory "+args[0])

	skeletonBase = path.Join(revelPkg.Dir, "skeleton")
	appDir = args[0]
	mustCopyDir(appDir, skeletonBase, map[string]interface{}{
		// app.conf
		"AppName": filepath.Base(appDir),
		"Secret":  genSecret(),
	})

	fmt.Fprintln(os.Stdout, "~ Your application is ready:\n~   ", appDir)
}

const alphaNumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func genSecret() string {
	chars := make([]byte, 64)
	for i := 0; i < 64; i++ {
		chars[i] = alphaNumeric[rand.Intn(len(alphaNumeric))]
	}
	return string(chars)
}
