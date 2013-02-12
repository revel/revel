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

func init() {
	RegisterPlugin(StartupPlugin{})
	RegisterPlugin(SessionPlugin{})
	RegisterPlugin(FlashPlugin{})
	RegisterPlugin(ValidationPlugin{})
	RegisterPlugin(InterceptorPlugin{})
	RegisterPlugin(I18nPlugin{})
}
