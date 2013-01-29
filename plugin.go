package revel

// An plugin that allows the user to inject behavior at various points in the request cycle.
type Plugin interface {
	// Called on server startup (and on each code reload).
	OnAppStart()
	// Called after the router has finished configuration.
	OnRoutesLoaded(router *Router)
	// Called before every request.
	BeforeRequest(c *Controller)
	// Called after every non-panicking request, before the Result has been applied.
	AfterRequest(c *Controller)
	// Called when a panic exits an action, with the recovered error value.
	OnException(c *Controller, err interface{})
	// Called after every request (panic or not), after the Result has been applied.
	Finally(c *Controller)
}

// It provides default (empty) implementations for all the required methods.
type EmptyPlugin struct{}

func (p EmptyPlugin) OnAppStart()                                {}
func (p EmptyPlugin) OnRoutesLoaded(router *Router)              {}
func (p EmptyPlugin) BeforeRequest(c *Controller)                {}
func (p EmptyPlugin) AfterRequest(c *Controller)                 {}
func (p EmptyPlugin) OnException(c *Controller, err interface{}) {}
func (p EmptyPlugin) Finally(c *Controller)                      {}

type PluginCollection []Plugin

var plugins PluginCollection

func RegisterPlugin(p Plugin) {
	plugins = append(plugins, p)
}

func (plugins PluginCollection) OnAppStart() {
	for _, p := range plugins {
		p.OnAppStart()
	}
}

func (plugins PluginCollection) OnRoutesLoaded(router *Router) {
	for _, p := range plugins {
		p.OnRoutesLoaded(router)
	}
}

func (plugins PluginCollection) BeforeRequest(c *Controller) {
	for _, p := range plugins {
		p.BeforeRequest(c)
	}
}

func (plugins PluginCollection) AfterRequest(c *Controller) {
	for _, p := range plugins {
		p.AfterRequest(c)
	}
}

func (plugins PluginCollection) OnException(c *Controller, err interface{}) {
	for _, p := range plugins {
		p.OnException(c, err)
	}
}

func (plugins PluginCollection) Finally(c *Controller) {
	for _, p := range plugins {
		p.Finally(c)
	}
}
