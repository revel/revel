package main

import (
	"github.com/robfig/revel"
	"github.com/robfig/revel/harness"
	"strconv"
)

var cmdRun = &Command{
	UsageLine: "run [import path] [run mode] [port]",
	Short:     "run a Revel application",
	Long: `
Run the Revel web application named by the given import path.

For example, to run the chat room sample application:

    revel run github.com/robfig/revel/samples/chat dev

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

You can set a port as an optional third parameter.  For example:

    revel run github.com/robfig/revel/samples/chat prod 8080`,
}

func init() {
	cmdRun.Run = runApp
}

func runApp(args []string) {
	if len(args) == 0 {
		errorf("No import path given.\nRun 'revel help run' for usage.\n")
	}

	// Determine the run mode.
	mode := "dev"
	if len(args) >= 2 {
		mode = args[1]
	}

	// Find and parse app.conf
	revel.Init(mode, args[0], "")
	revel.LoadMimeConfig()

	// Determine the override port, if any.
	port := revel.HttpPort
	if len(args) == 3 {
		var err error
		if port, err = strconv.Atoi(args[2]); err != nil {
			errorf("Failed to parse port as integer: %s", args[2])
		}
	}

	revel.INFO.Printf("Running %s (%s) in %s mode\n", revel.AppName, revel.ImportPath, mode)
	revel.TRACE.Println("Base path:", revel.BasePath)

	// If the app is run in "watched" mode, use the harness to run it.
	if revel.Config.BoolDefault("watch", true) && revel.Config.BoolDefault("watch.code", true) {
		revel.HttpPort = port
		harness.NewHarness().Run() // Never returns.
	}

	// Else, just build and run the app.
	app, err := harness.Build()
	if err != nil {
		errorf("Failed to build app: %s", err)
	}
	app.Port = port
	app.Cmd().Run()
}
