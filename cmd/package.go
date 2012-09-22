package main

import (
	"fmt"
	"github.com/robfig/revel"
	"github.com/robfig/revel/harness"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

var cmdPackage = &Command{
	UsageLine: "package [import path]",
	Short:     "package a Revel application (e.g. for deployment)",
	Long: `~
~ Package the Revel web application named by the given import path.
~ This allows it to be deployed and run on a machine that lacks a Go installation.
~
~ For example:
~
~     rev package github.com/robfig/revel/samples/chat
~`,
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
	rev.Init("", appImportPath, "")

	binName, reverr := harness.Build()
	panicOnError(reverr, "Failed to build")

	// Start collecting stuff in a temp directory.
	tmpDir, err := ioutil.TempDir("", rev.AppName)
	panicOnError(err, "Failed to get temp dir")

	// Included are:
	// - run scripts
	// - binary
	// - revel
	// - app

	// Revel and the app are in a directory structure mirroring import path
	tmpRevelPath := path.Join(tmpDir, filepath.FromSlash(rev.REVEL_IMPORT_PATH))
	mustCopyFile(path.Join(tmpDir, filepath.Base(binName)), binName)
	mustCopyDir(path.Join(tmpRevelPath, "conf"), path.Join(rev.RevelPath, "conf"), nil)
	mustCopyDir(path.Join(tmpRevelPath, "templates"), path.Join(rev.RevelPath, "templates"), nil)
	mustCopyDir(path.Join(tmpDir, filepath.FromSlash(appImportPath)), rev.BasePath, nil)

	tmplData := map[string]interface{}{
		"BinName":    filepath.Base(binName),
		"ImportPath": appImportPath,
	}

	mustRenderTemplate(
		path.Join(tmpDir, "run.sh"),
		path.Join(rev.RevelPath, "cmd", "package_run.sh.template"),
		tmplData)

	mustRenderTemplate(
		path.Join(tmpDir, "run.bat"),
		path.Join(rev.RevelPath, "cmd", "package_run.bat.template"),
		tmplData)

	// Create the zip file.
	zipName := mustZipDir(rev.AppName+".zip", tmpDir)

	fmt.Println("~ Your archive is ready:", zipName)
}
