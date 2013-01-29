package main

import (
	"archive/zip"
	"fmt"
	"github.com/robfig/revel"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

// Use a wrapper to differentiate logged panics from unexpected ones.
type LoggedError struct{ error }

func panicOnError(err error, msg string) {
	if revErr, ok := err.(*revel.Error); (ok && revErr != nil) || (!ok && err != nil) {
		fmt.Fprintf(os.Stderr, "Abort: %s: %s\n", msg, err)
		panic(LoggedError{err})
	}
}

func mustCopyFile(destFilename, srcFilename string) {
	destFile, err := os.Create(destFilename)
	panicOnError(err, "Failed to create file "+destFilename)

	srcFile, err := os.Open(srcFilename)
	panicOnError(err, "Failed to open file "+srcFilename)

	_, err = io.Copy(destFile, srcFile)
	panicOnError(err,
		fmt.Sprintf("Failed to copy data from %s to %s", srcFile.Name(), destFile.Name()))

	err = destFile.Close()
	panicOnError(err, "Failed to close file "+destFile.Name())

	err = srcFile.Close()
	panicOnError(err, "Failed to close file "+srcFile.Name())
}

func mustRenderTemplate(destPath, srcPath string, data map[string]interface{}) {
	tmpl, err := template.ParseFiles(srcPath)
	panicOnError(err, "Failed to parse template "+srcPath)

	f, err := os.Create(destPath)
	panicOnError(err, "Failed to create "+destPath)

	err = tmpl.Execute(f, data)
	panicOnError(err, "Failed to render template "+srcPath)

	err = f.Close()
	panicOnError(err, "Failed to close "+f.Name())
}

// copyDir copies a directory tree over to a new directory.  Any files ending in
// ".template" are treated as a Go template and rendered using the given data.
// Additionally, the trailing ".template" is stripped from the file name.
// Also, dot files and dot directories are skipped.
func mustCopyDir(destDir, srcDir string, data map[string]interface{}) error {
	return filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		// Get the relative path from the source base, and the corresponding path in
		// the dest directory.
		relSrcPath := strings.TrimLeft(srcPath[len(srcDir):], string(os.PathSeparator))
		destPath := path.Join(destDir, relSrcPath)

		// Skip dot files and dot directories.
		if strings.HasPrefix(relSrcPath, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create a subdirectory if necessary.
		if info.IsDir() {
			err := os.MkdirAll(path.Join(destDir, relSrcPath), 0777)
			if !os.IsExist(err) {
				panicOnError(err, "Failed to create directory")
			}
			return nil
		}

		// If this file ends in ".template", render it as a template.
		if strings.HasSuffix(relSrcPath, ".template") {
			mustRenderTemplate(destPath[:len(destPath)-len(".template")], srcPath, data)
			return nil
		}

		// Else, just copy it over.
		mustCopyFile(destPath, srcPath)
		return nil
	})
}

func mustZipDir(destFilename, srcDir string) string {
	zipFile, err := os.Create(destFilename)
	panicOnError(err, "Failed to create zip file")

	w := zip.NewWriter(zipFile)
	filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		// Ignore directories (they are represented by the path of written entries).
		if info.IsDir() {
			return nil
		}

		relSrcPath := strings.TrimLeft(srcPath[len(srcDir):], string(os.PathSeparator))

		f, err := w.Create(relSrcPath)
		panicOnError(err, "Failed to create zip entry")

		srcFile, err := os.Open(srcPath)
		panicOnError(err, "Failed to read source file")

		_, err = io.Copy(f, srcFile)
		panicOnError(err, "Failed to copy")

		return nil
	})

	err = w.Close()
	panicOnError(err, "Failed to close archive")

	err = zipFile.Close()
	panicOnError(err, "Failed to close zip file")

	return zipFile.Name()
}
