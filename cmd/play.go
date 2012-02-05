// The command line tool for running Play apps.
// Presently it does nothing but run the harness / sample app.
package main

import (
	"flag"
	"fmt"
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
	play.Init(args[0])
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
