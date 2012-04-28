package rev

// An plugin that allows the user to inject behavior at various points in the request cycle.
type Plugin interface {
	OnAppStart()
	BeforeRequest(c *Controller)
	AfterRequest(c *Controller)
	OnException(c *Controller, err interface{})
}

// It provides default (empty) implementations for all the required methods.
type EmptyPlugin struct{}

func (p EmptyPlugin) OnAppStart()                                {}
func (p EmptyPlugin) BeforeRequest(c *Controller)                {}
func (p EmptyPlugin) AfterRequest(c *Controller)                 {}
func (p EmptyPlugin) OnException(c *Controller, err interface{}) {}

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
