package main

import (
	"github.com/robfig/revel"
	"github.com/robfig/revel/harness"
	"log"
)

var cmdRun = &Command{
	UsageLine: "run [import path] [run mode]",
	Short:     "run a Revel application",
	Long: `
Run the Revel web application named by the given import path.

For example, to run the chat room sample application:

    revel run github.com/robfig/revel/samples/chat dev

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".`,
}

func init() {
	cmdRun.Run = runApp
}

func runApp(args []string) {
	if len(args) == 0 {
		errorf("No import path given.\nRun 'revel help run' for usage.\n")
	}

	mode := "dev"
	if len(args) == 2 {
		mode = args[1]
	}

	// Find and parse app.conf
	rev.Init(mode, args[0], "")
	log.Printf("Running %s (%s) in %s mode\n", rev.AppName, rev.ImportPath, mode)
	rev.TRACE.Println("Base path:", rev.BasePath)

	// If the app is run in "watched" mode, use the harness to run it.
	if rev.Config.BoolDefault("server.watcher", true) {
		harness.NewHarness().Run() // Never returns.
	}

	// Else, just build and run the app.
	app, err := harness.Build()
	if err != nil {
		errorf("Failed to build app: %s", err)
	}
	app.Cmd().Run()
}
