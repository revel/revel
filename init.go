package revel

func init() {
	RegisterPlugin(SessionPlugin{})
	RegisterPlugin(FlashPlugin{})
	RegisterPlugin(ValidationPlugin{})
	RegisterPlugin(InterceptorPlugin{})
	RegisterPlugin(I18nPlugin{})
}
