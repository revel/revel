// The command line tool for running Revel apps.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)

// Cribbed from the genius organization of the "go" command.
type Command struct {
	Run                    func(args []string)
	UsageLine, Short, Long string
}

func (cmd *Command) Name() string {
	name := cmd.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

var commands = []*Command{
	cmdRun,
	cmdNew,
	cmdClean,
	cmdPackage,
	cmdAutoTest,
}

func main() {
	fmt.Fprintf(os.Stdout, header)
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 || args[0] == "help" {
		if len(args) > 1 {
			for _, cmd := range commands {
				if cmd.Name() == args[1] {
					tmpl(os.Stdout, helpTemplate, cmd)
					return
				}
			}
		}
		usage()
	}

	// Commands use panic to abort execution when something goes wrong.
	// Panics are logged at the point of error.  Ignore those.
	defer func() {
		if err := recover(); err != nil {
			if _, ok := err.(LoggedError); !ok {
				// This panic was not expected / logged.
				panic(err)
			}
		}
	}()

	for _, cmd := range commands {
		if cmd.Name() == args[0] {
			cmd.Run(args[1:])
			return
		}
	}

	errorf("unknown command %q\nRun 'revel help' for usage.\n", args[0])
}

func errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

const header = `~
~ revel! http://robfig.github.com/revel
~
`

const usageTemplate = `usage: revel command [arguments]

The commands are:
{{range .}}
    {{.Name | printf "%-11s"}} {{.Short}}{{end}}

Use "revel help [command]" for more information.
`

var helpTemplate = `usage: revel {{.UsageLine}}
{{.Long}}
`

func usage() {
	tmpl(os.Stderr, usageTemplate, commands)
	os.Exit(2)
}

func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	template.Must(t.Parse(text))
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}
