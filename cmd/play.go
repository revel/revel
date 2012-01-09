// The command line tool for running Play apps.
// Presently it does nothing but run the harness / sample app.
//
// GB options
// target: play
package main

import (
	"play/harness"
)

func main() {
	harness.Run()
}
