package revel

// Allows code to get run on startup without implementing a full-blown plugin.
func OnAppStart(f func()) {
	println("OnAppStart")
	hooks = append(hooks, f)
}

var hooks []func()

type StartupPlugin struct {
	EmptyPlugin
}

func (p StartupPlugin) OnAppStart() {
	println("Running OnAppStarts")
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
