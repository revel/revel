package main

import (
	"fmt"
	"github.com/robfig/revel"
	"github.com/robfig/revel/harness"
	"os"
	"os/exec"
	"runtime"
	"strconv"
)

var cmdDebug = &Command{
	UsageLine: "debug [import path] [port]",
	Short:     "debug a Revel application",
	Long: `
Debug the Revel web application named by the given import path.

For example, to debug the chat room sample application:

    revel debug github.com/robfig/revel/samples/chat

Debug implies the "dev" run mode.

You can set a port as an optional second parameter.  For example:

    revel debug github.com/robfig/revel/samples/chat 8080`,
}

func init() {
	cmdDebug.Run = debugApp
}

func debugApp(args []string) {
	if runtime.GOOS != "linux" {
		errorf("Debugging is only supported on Linux.\n")
	}

	if len(args) == 0 {
		errorf("No import path given.\nRun 'revel help debug' for usage.\n")
	}

	mode := "dev"

	// Find and parse app.conf
	revel.Init(mode, args[0], "")
	revel.LoadMimeConfig()

	// Determine the override port, if any.
	port := revel.HttpPort
	if len(args) == 2 {
		var err error
		if port, err = strconv.Atoi(args[1]); err != nil {
			errorf("Failed to parse port as integer: %s", args[1])
		}
	} else if len(args) > 2 {
		errorf("Too many options.\nRun 'revel help run' for usage.\n")
	}

	revel.INFO.Printf("Debugging %s (%s)\n", revel.AppName, revel.ImportPath)
	revel.TRACE.Println("Base path:", revel.BasePath)

	app, errHarness := harness.Build()
	if errHarness != nil {
		errorf("Failed to build app: %s", errHarness)
	}

	// Debugging relies on the user having "gdb" installed.
	gdbPath, err := exec.LookPath("gdb")
	if err != nil {
		revel.ERROR.Fatalf("GDB executable not found in PATH.")
	}

	// Launch gdb.
	cmd := exec.Command(gdbPath, "--args", app.BinaryPath,
		fmt.Sprintf("-port=%d", port),
		fmt.Sprintf("-importPath=%s", revel.ImportPath),
		fmt.Sprintf("-runMode=%s", revel.RunMode))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	revel.TRACE.Println("Exec app:", cmd.Args)
	err = cmd.Start()
	if err != nil {
		revel.ERROR.Fatalf("Failed to start gdb: %s", err)
	}
	err = cmd.Wait()
	if err != nil {
		revel.ERROR.Fatalf("Error while debugging: %v\n", err)
	}
}
