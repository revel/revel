package harness

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/robfig/revel"
	"io"
	"os"
	"os/exec"
	"time"
)

// App contains the configuration for running a Revel app.  (Not for the app itself)
// Its only purpose is constructing the command to execute.
type App struct {
	BinaryPath string // Path to the app executable
	Port       int    // Port to pass as a command line argument.
	cmd        AppCmd // The last cmd returned.
}

func NewApp(binPath string) *App {
	return &App{BinaryPath: binPath}
}

// Return a command to run the app server using the current configuration.
func (a *App) Cmd() AppCmd {
	a.cmd = NewAppCmd(a.BinaryPath, a.Port)
	return a.cmd
}

// Kill the last app command returned.
func (a *App) Kill() {
	a.cmd.Kill()
}

// AppCmd manages the running of a Revel app server.
// It requires revel.Init to have been called previously.
type AppCmd struct {
	*exec.Cmd
}

func NewAppCmd(binPath string, port int) AppCmd {
	cmd := exec.Command(binPath,
		fmt.Sprintf("-port=%d", port),
		fmt.Sprintf("-importPath=%s", revel.ImportPath),
		fmt.Sprintf("-runMode=%s", revel.RunMode))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return AppCmd{cmd}
}

// Start the app server, and wait until it is ready to serve requests.
func (cmd AppCmd) Start() error {
	listeningWriter := startupListeningWriter{os.Stdout, make(chan bool)}
	cmd.Stdout = listeningWriter
	revel.TRACE.Println("Exec app:", cmd.Path, cmd.Args)
	if err := cmd.Cmd.Start(); err != nil {
		revel.ERROR.Fatalln("Error running:", err)
	}

	select {
	case <-cmd.waitChan():
		return errors.New("revel/harness: app died")

	case <-time.After(30 * time.Second):
		cmd.Kill()
		return errors.New("revel/harness: app timed out")

	case <-listeningWriter.notifyReady:
		return nil
	}
	panic("Impossible")
}

// Run the app server inline.  Never returns.
func (cmd AppCmd) Run() {
	revel.TRACE.Println("Exec app:", cmd.Path, cmd.Args)
	if err := cmd.Cmd.Run(); err != nil {
		revel.ERROR.Fatalln("Error running:", err)
	}
}

// Terminate the app server if it's running.
func (cmd AppCmd) Kill() {
	if cmd.Cmd != nil && (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) {
		revel.TRACE.Println("Killing revel server pid", cmd.Process.Pid)
		err := cmd.Process.Kill()
		if err != nil {
			revel.ERROR.Fatalln("Failed to kill revel server:", err)
		}
	}
}

// Return a channel that is notified when Wait() returns.
func (cmd AppCmd) waitChan() <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		cmd.Wait()
		ch <- struct{}{}
	}()
	return ch
}

// A io.Writer that copies to the destination, and listens for "Listening on.."
// in the stream.  (Which tells us when the revel server has finished starting up)
// This is super ghetto, but by far the simplest thing that should work.
type startupListeningWriter struct {
	dest        io.Writer
	notifyReady chan bool
}

func (w startupListeningWriter) Write(p []byte) (n int, err error) {
	if w.notifyReady != nil && bytes.Contains(p, []byte("Listening")) {
		w.notifyReady <- true
		w.notifyReady = nil
	}
	return w.dest.Write(p)
}
