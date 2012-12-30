package controllers

import "github.com/robfig/revel"

type Application struct {
	*rev.Controller
}

func (c Application) Index() rev.Result {
	// Localization information
	c.RenderArgs["acceptLanguageHeader"] = c.Request.Header.Get("Accept-Language")
	currentLocale := c.Args["currentLocale"].(string)
	c.RenderArgs["controllerCurrentLocale"] = currentLocale

	// Controller-resolves messages
	c.RenderArgs["controllerGreeting"] = c.Message("greeting")
	c.RenderArgs["controllerGreetingName"] = c.Message("greeting.name")
	c.RenderArgs["controllerGreetingSuffix"] = c.Message("greeting.suffix")
	c.RenderArgs["controllerGreetingFull"] = c.Message("greeting.full")
	c.RenderArgs["controllerGreetingWithArgument"] = c.Message("greeting.full.name", "Steve Buscemi")

	return c.Render()
}
