package main

import (
	"github.com/robfig/revel"
	"github.com/robfig/revel/harness"
	"log"
)

var cmdAutoTest = &Command{
	UsageLine: "auto-test [import path] [run mode]",
	Short:     "run all tests from the command-line",
	Long: `
Run all tests for the Revel app named by the given import path.

For example, to run the booking sample application's tests:

    revel auto-test github.com/robfig/revel/samples/booking dev

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".`,
}

func init() {
	cmdAutoTest.Run = autoTestApp
}

func autoTestApp(args []string) {
	if len(args) == 0 {
		errorf("No import path given.\nRun 'revel help run' for usage.\n")
	}

	mode := "dev"
	if len(args) == 2 {
		mode = args[1]
	}

	// Find and parse app.conf
	rev.Init(mode, args[0], "")
	log.Printf("Testing %s (%s) in %s mode\n", rev.AppName, rev.ImportPath, mode)
	rev.TRACE.Println("Base path:", rev.BasePath)

	cmd := harness.StartApp(false)

	// The app is now running.  Execute all functional tests.
	var succeeded, failed int
	for _, test := range rev.FunctionalTests {
		x, y := rev.RunTestSuite(test)
		succeeded += x
		failed += y
	}

	cmd.Process.Kill()

	log.Print("Test Summary")
	log.Print("============")
	log.Print("Passed:", succeeded)
	log.Print("Failed:", failed)

	if failed >= 0 {
		errorf("Failure.")
	}
}
