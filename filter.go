package revel

type FilterChain []Filter

type Filter interface {
	Call(c *Controller, chain FilterChain)
}

type InitializingFilter interface {
	Filter
	OnAppStart()
}

var Filters = FilterChain{
	PanicFilter{},
	ParamsFilter{},
	RouterFilter{},
	SessionFilter{},
	FlashFilter{},
	ValidationFilter{},
	I18nFilter{},
	InterceptorFilter{},
	ActionInvoker{},
}

// NilFilter and NilChain are helpful in writing filter tests.
var (
	NilFilter nilFilter
	NilChain  = FilterChain{NilFilter}
)

type nilFilter struct{}

func (f nilFilter) Call(_ *Controller, _ FilterChain) {}
