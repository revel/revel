package app

import "github.com/robfig/revel"

func init() {
	// Filters is the default set of global filters.
	revel.Filters = []Filter{
		PanicFilter,             // Recover from panics and display an error page instead.
		RouterFilter,            // Use the routing table to select the right Action
		FilterConfiguringFilter, // A hook for adding or removing per-Action filters.
		ParamsFilter,            // Parse parameters into Controller.Params.
		SessionFilter,           // Restore and write the session cookie.
		FlashFilter,             // Restore and write the flash cookie.
		ValidationFilter,        // Restore kept validation errors and save new ones from cookie.
		I18nFilter,              // Resolve the requested language
		InterceptorFilter,       // Run interceptors around the action.
		ActionInvoker,           // Invoke the action.
	}
}
