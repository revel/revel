// This program generates per-file godoc.
//
// go run docfile.go -templates ~/Dropbox/Public/revel/docs/godoc -out ~/Dropbox/Public/revel/docs  ~/code/gocode/src/github.com/robfig/revel/*.go
package main

import (
	"bytes"
	"os/exec"
	"flag"
	"path"
	"path/filepath"
	"log"
	"os"
	"strings"
)

var (
	templates = flag.String("templates", "", "Path to the package.html template")
	out = flag.String("out", "out", "Path to the generated stuff")
)

func main() {
	flag.Parse()

	// Find godoc
	gdPath, err := exec.LookPath("godoc")
	if err != nil {
		log.Fatalln(err)
	}

	// Create a temp directory to use for linking the files
	tempDirPath := path.Join(os.TempDir(), "docfile")

	// Decide where the generated files are going.
	docPath := path.Join(*out, "godoc")
	srcPath := path.Join(*out, "src")

	// Link each file in turn.
	for _, filename := range flag.Args() {
		if !strings.HasSuffix(filename, ".go") || strings.HasSuffix(filename, "_test.go"){
			continue
		}
		log.Println("Processing", filename)

		filename, err := filepath.Abs(filename)
		must(err)

		// Delete, recreate the tempdir, and link in the source file.
		basename := path.Base(filename)
		must(os.RemoveAll(tempDirPath))
		must(os.MkdirAll(tempDirPath, 0777))
		must(os.Symlink(filename, path.Join(tempDirPath, basename)))

		// Get the target html file ready.
		htmlname := basename[:len(basename)-3] + ".html"
		log.Println("Writing", path.Join(docPath, htmlname))
		htmlFile, err := os.Create(path.Join(docPath, htmlname))
		must(err)

		// Print the docs to html.
		cmd := exec.Command(gdPath, "-html", "-templates", *templates, tempDirPath)
		b, err := cmd.CombinedOutput()
		must(err)
		_, err = htmlFile.Write(replace(b, basename, htmlname))
		must(err)
		must(htmlFile.Close())

		// Now print the source to html.
		log.Println("Writing", path.Join(srcPath, htmlname))
		srcFile, err := os.Create(path.Join(srcPath, htmlname))
		must(err)
		cmd = exec.Command(gdPath, "-src", "-html", "-templates", *templates, tempDirPath)
		b, err = cmd.CombinedOutput()
		must(err)
		_, err = srcFile.Write(replace(b, basename, htmlname))
		must(err)
		must(srcFile.Close())
	}

}

func replace(b []byte, basename, htmlname string) []byte {
	b = bytes.Replace(b, []byte("%%TITLE%%"),
		[]byte(strings.Title(basename[:len(basename)-3])), -1)
	b = bytes.Replace(b, []byte("/target/"+basename),
		[]byte("../src/"+htmlname), -1)
	return b
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
