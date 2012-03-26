// The command line tool for running Play apps.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"play"
	"play/harness"
)

func main() {
	fmt.Fprintf(os.Stdout, header)
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 || len(args) > 2 || args[0] == "help" {
		usage()
	}

	mode := play.DEV
	if len(args) == 2 && args[1] == "prod" {
		mode = play.PROD
	}

	// Find and parse app.conf
	play.Init(args[0], mode)
	log.Printf("Running app (%s): %s (%s)\n", mode, play.AppName, play.BasePath)

	harness.Run(mode)
}

const header = `~
~ go play! http://www.github.com/robfig/go-play
~
`

const usageText = `~ Usage: play import_path [mode]
`

func usage() {
	fmt.Fprintf(os.Stderr, usageText)
	os.Exit(2)
}
