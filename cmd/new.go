package main

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
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
	_, err := os.Open(args[0])
	if err == nil {
		fmt.Fprintf(os.Stderr, "~ Abort: Directory %s already exists.", args[0])
		return
	}

	revelPkg, err := build.Import("github.com/robfig/revel", "", build.FindOnly)
	if err != nil {
		fmt.Fprintln(os.Stderr, "~ Failed to find revel code.")
		return
	}

	err = os.MkdirAll(args[0], 0777)
	if err != nil {
		fmt.Fprintln(os.Stderr, "~ Abort: Failed to create directory:", err)
		return
	}

	skeletonBase = path.Join(revelPkg.Dir, "skeleton")
	appDir, err = filepath.Abs(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "~ Abort: Failed to get absolute directory:", err)
		return
	}

	err = filepath.Walk(skeletonBase, copySkeleton)
	if err != nil {
		fmt.Fprintln(os.Stderr, "~ Failed to copy skeleton:", err)
	}

	fmt.Fprintln(os.Stdout, "~ Your application is ready:\n~   ", appDir)
}

// copySkeleton copies the skeleton app tree over to a new directory.
func copySkeleton(skelPath string, skelInfo os.FileInfo, err error) error {
	// Get the relative path from the skeleton base, and the corresponding path in
	// the app directory.
	relSkelPath := strings.TrimLeft(skelPath[len(skeletonBase):], string(os.PathSeparator))
	appFile := path.Join(appDir, relSkelPath)

	if len(relSkelPath) == 0 {
		return nil
	}

	// Create a subdirectory if necessary.
	if skelInfo.IsDir() {
		err := os.Mkdir(path.Join(appDir, relSkelPath), 0777)
		if err != nil {
			fmt.Fprintln(os.Stderr, "~ Failed to create directory:", err)
			return err
		}
		return nil
	}

	// If this is app.conf, we have to render it as a template.
	if relSkelPath == "conf/app.conf" {
		tmpl, err := template.ParseFiles(skelPath)
		if err != nil || tmpl == nil {
			fmt.Fprintln(os.Stderr, "Failed to parse skeleton app.conf as a template:", err)
			return err
		}

		f, err := os.Create(appFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to create app.conf:", err)
			return err
		}

		err = tmpl.Execute(f, map[string]string{
			"AppName": filepath.Base(appDir),
			"Secret":  genSecret(),
		})

		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to render template:", err)
			return err
		}

		err = f.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to close app.conf:", err)
			return err
		}
		return nil
	}

	// Copy over the files.
	skelBytes, err := ioutil.ReadFile(skelPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "~ Failed to read file:", err)
		return err
	}

	err = ioutil.WriteFile(appFile, skelBytes, 0666)
	if err != nil {
		fmt.Fprintln(os.Stderr, "~ Failed to write file:", err)
		return err
	}
	return nil
}

const alphaNumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func genSecret() string {
	chars := make([]byte, 64)
	for i := 0; i < 64; i++ {
		chars[i] = alphaNumeric[rand.Intn(len(alphaNumeric))]
	}
	return string(chars)
}
