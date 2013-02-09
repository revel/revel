package main

import (
	"fmt"
	"github.com/robfig/revel"
	"github.com/robfig/revel/harness"
	"os"
	"path/filepath"
)

var cmdBuild = &Command{
	UsageLine: "build [import path] [target path]",
	Short:     "build a Revel application (e.g. for deployment)",
	Long: `
Build the Revel web application named by the given import path.
This allows it to be deployed and run on a machine that lacks a Go installation.

WARNING: The target path will be completely deleted, if it already exists!

For example:

    revel build github.com/robfig/revel/samples/chat /tmp/chat
`,
}

func init() {
	cmdBuild.Run = buildApp
}

func buildApp(args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "%s\n%s", cmdBuild.UsageLine, cmdBuild.Long)
		return
	}

	appImportPath, destPath := args[0], args[1]
	if !revel.Initialized {
		revel.Init("", appImportPath, "")
	}

	os.RemoveAll(destPath)
	os.MkdirAll(destPath, 0777)

	app, reverr := harness.Build()
	panicOnError(reverr, "Failed to build")

	// Included are:
	// - run scripts
	// - binary
	// - revel
	// - app

	// Revel and the app are in a directory structure mirroring import path
	srcPath := filepath.Join(destPath, "src")
	tmpRevelPath := filepath.Join(srcPath, filepath.FromSlash(revel.REVEL_IMPORT_PATH))
	mustCopyFile(filepath.Join(destPath, filepath.Base(app.BinaryPath)), app.BinaryPath)
	mustCopyDir(filepath.Join(tmpRevelPath, "conf"), filepath.Join(revel.RevelPath, "conf"), nil)
	mustCopyDir(filepath.Join(tmpRevelPath, "templates"), filepath.Join(revel.RevelPath, "templates"), nil)
	mustCopyDir(filepath.Join(srcPath, filepath.FromSlash(appImportPath)), revel.BasePath, nil)

	tmplData := map[string]interface{}{
		"BinName":    filepath.Base(app.BinaryPath),
		"ImportPath": appImportPath,
	}

	mustRenderTemplate(
		filepath.Join(destPath, "run.sh"),
		filepath.Join(revel.RevelPath, "cmd", "package_run.sh.template"),
		tmplData)

	mustRenderTemplate(
		filepath.Join(destPath, "run.bat"),
		filepath.Join(revel.RevelPath, "cmd", "package_run.bat.template"),
		tmplData)
}
