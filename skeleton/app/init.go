package app

import "github.com/robfig/revel"

func init() {
	revel.Filters = revel.FilterChain{
		revel.PanicFilter{},
		revel.ParamsFilter{},
		revel.RouterFilter{},
		revel.SessionFilter{},
		revel.FlashFilter{},
		revel.ValidationFilter{},
		revel.I18nFilter{},
		revel.InterceptorFilter{},
		revel.ActionInvoker{},
	}
}
