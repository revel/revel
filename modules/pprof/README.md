Revel pprof module
============

#### How to use:

1. Open your app.conf file and add the following line:  
`module.pprof=github.com/revel/revel/modules/pprof`  
This will enable the pprof module.

2. Next, open your routes file and add:  
`module:pprof` **Note:** Do not change these routes. The pprof command-line tool by default assumes these routes.

Congrats! You can now profile your application. To use the web interface, visit `http://<host>:<port>/debug/pprof`. You can also use the `go tool pprof` command to profile your application and get a little deeper. Use the command by running `go tool pprof <binary> http://<host>:<port>`. For example, if you modified the booking sample, you would run: `go tool pprof $GOPATH/bin/booking http://localhost:9000` (assuming you used the default `revel run` arguments.

The command-line tool will take a 30-second CPU profile, and save the results to a temporary file in your `$HOME/pprof` directory. You can reference this file at a later time by using the same command as above, but by specifying the filename instead of the server address.

In order to fully utilize the command-line tool, you may need the graphviz utilities. If you're on OS X, you can install these easily using Homebrew: `brew install graphviz`.

To read more about profiling Go programs, here is some reading material: [The Go Blog](http://blog.golang.org/profiling-go-programs) : [net/pprof package documentation](http://golang.org/pkg/net/http/pprof/)