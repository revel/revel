package revel

import (
	"sort"
)

// The list of startup hooks
type (
	// The startup hooks structure
	RevelHook struct {
		order int    // The order
		f     func() // The function to call
	}

	RevelHooks []RevelHook
)

var (
	// The local instance of the list
	startupHooks RevelHooks

	// The instance of the list
	shutdownHooks RevelHooks
)

// Called to run the hooks
func (r RevelHooks) Run() {
	serverLogger.Infof("There is %d hooks need to run ...", len(r))
	sort.Sort(r)
	for i, hook := range r {
		utilLog.Infof("Run the %d hook ...", i+1)
		hook.f()
	}
}

// Adds a new function hook, using the order
func (r RevelHooks) Add(fn func(), order ...int) RevelHooks {
	o := 1
	if len(order) > 0 {
		o = order[0]
	}
	return append(r, RevelHook{order: o, f: fn})
}

// Sorting function
func (slice RevelHooks) Len() int {
	return len(slice)
}

// Sorting function
func (slice RevelHooks) Less(i, j int) bool {
	return slice[i].order < slice[j].order
}

// Sorting function
func (slice RevelHooks) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// OnAppStart registers a function to be run at app startup.
//
// The order you register the functions will be the order they are run.
// You can think of it as a FIFO queue.
// This process will happen after the config file is read
// and before the server is listening for connections.
//
// Ideally, your application should have only one call to init() in the file init.go.
// The reason being that the call order of multiple init() functions in
// the same package is undefined.
// Inside of init() call revel.OnAppStart() for each function you wish to register.
//
// Example:
//
//      // from: yourapp/app/controllers/somefile.go
//      func InitDB() {
//          // do DB connection stuff here
//      }
//
//      func FillCache() {
//          // fill a cache from DB
//          // this depends on InitDB having been run
//      }
//
//      // from: yourapp/app/init.go
//      func init() {
//          // set up filters...
//
//          // register startup functions
//          revel.OnAppStart(InitDB)
//          revel.OnAppStart(FillCache)
//      }
//
// This can be useful when you need to establish connections to databases or third-party services,
// setup app components, compile assets, or any thing you need to do between starting Revel and accepting connections.
//
func OnAppStart(f func(), order ...int) {
	startupHooks = startupHooks.Add(f, order...)
}

// Called to add a new stop hook
func OnAppStop(f func(), order ...int) {
	shutdownHooks = shutdownHooks.Add(f, order...)
}
