// The command line tool for running Play apps.
// Presently it does nothing but run the harness / sample app.
//
// GB options
// target: play
package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"os"

	"play"
	"play/harness"
)

func main() {
	fmt.Fprintf(os.Stdout, header)
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 || args[0] == "help" {
		usage()
	}

	// Find and parse application.yaml
	appDirPath, err := filepath.Abs(args[0])
	if err != nil {
		play.LOG.Fatalln(err.Error())
	}

	play.Init(appDirPath)
	play.LOG.Printf("Running app: %s (%s)\n", play.AppName, play.BasePath)

	harness.Run()
}

const header = `~
~ go play! http://www.github.com/robfig/go-play
~
`

const usageText = `~ Usage: play [app_path]
`

func usage() {
	fmt.Fprintf(os.Stderr, usageText)
	os.Exit(2)
}
