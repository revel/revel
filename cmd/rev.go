// The command line tool for running Revel apps.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/robfig/revel"
	"github.com/robfig/revel/harness"
)

func main() {
	fmt.Fprintf(os.Stdout, header)
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 || len(args) > 2 || args[0] == "help" {
		usage()
	}

	mode := rev.DEV
	if len(args) == 2 && args[1] == "prod" {
		mode = rev.PROD
	}

	// Find and parse app.conf
	rev.Init(args[0], mode)
	log.Printf("Running app (%s): %s (%s)\n", mode, rev.AppName, rev.BasePath)

	harness.Run(mode)
}

const header = `~
~ revel! http://www.github.com/robfig/revel
~
`

const usageText = `~ Usage: rev import_path [mode]
`

func usage() {
	fmt.Fprintf(os.Stderr, usageText)
	os.Exit(2)
}
