package main

import (
	"github.com/robfig/revel"
	"github.com/robfig/revel/harness"
	"log"
)

var cmdRun = &Command{
	UsageLine: "run [import path] [run mode]",
	Short:     "run a Revel application",
	Long: `~
~ Run the Revel web application named by the given import path.
~
~ For example, to run the chat room sample application:
~
~     rev run github.com/robfig/revel/samples/chat
~
~ The run mode is used to select which set of app.conf configuration should
~ apply and may be used to determine logic in the application itself.
~
~ Run mode defaults to "dev". To run in production mode, specify "prod"`,
}

func init() {
	cmdRun.Run = runApp
}

func runApp(args []string) {
	if len(args) == 0 {
		errorf("~ No import path given.\nRun 'rev help run' for usage.\n")
	}

	// TODO: Why can't it be any arbitrary string again?
	mode := rev.DEV
	if len(args) == 2 && args[1] == rev.PROD {
		mode = rev.PROD
	}

	// Find and parse app.conf
	rev.Init(args[0], mode)
	log.Printf("Running app (%s): %s (%s)\n", mode, rev.AppName, rev.BasePath)

	harness.Run(mode)
}
