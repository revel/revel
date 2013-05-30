package revel

// A simple hook to run code on startup without having to implement a plugin.
func OnAppStart(f func()) {
	hooks = append(hooks, f)
}

var hooks []func()

type StartupPlugin struct {
	EmptyPlugin
}

func (p StartupPlugin) OnAppStart() {
	for _, hook := range hooks {
		hook()
	}
}
