package revel

type Filter func(c *Controller, filterChain []Filter)

// Filters is the default set of global filters.
// It may be set by the application on initialization.
var Filters = []Filter{
	PanicFilter,
	RouterFilter,
	FilterConfiguringFilter,
	ParamsFilter,
	SessionFilter,
	FlashFilter,
	ValidationFilter,
	I18nFilter,
	InterceptorFilter,
	ActionInvoker,
}

// NilFilter and NilChain are helpful in writing filter tests.
var (
	NilFilter = func(_ *Controller, _ []Filter) {}
	NilChain  = []Filter{NilFilter}
)
