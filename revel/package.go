package main

import (
	"fmt"
	"github.com/robfig/revel"
	"io/ioutil"
	"os"
	"path/filepath"
)

var cmdPackage = &Command{
	UsageLine: "package [import path]",
	Short:     "package a Revel application (e.g. for deployment)",
	Long: `
Package the Revel web application named by the given import path.
This allows it to be deployed and run on a machine that lacks a Go installation.

For example:

    revel package github.com/robfig/revel/samples/chat
`,
}

func init() {
	cmdPackage.Run = packageApp
}

func packageApp(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, cmdPackage.Long)
		return
	}

	appImportPath := args[0]
	revel.Init("", appImportPath, "")

	// Remove the archive if it already exists.
	destFile := filepath.Base(revel.BasePath) + ".tar.gz"
	os.Remove(destFile)

	// Collect stuff in a temp directory.
	tmpDir, err := ioutil.TempDir("", filepath.Base(revel.BasePath))
	panicOnError(err, "Failed to get temp dir")

	buildApp([]string{args[0], tmpDir})

	// Create the zip file.
	archiveName := mustTarGzDir(destFile, tmpDir)

	fmt.Println("Your archive is ready:", archiveName)
}
