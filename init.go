package rev

func init() {
	RegisterPlugin(SessionPlugin{})
	RegisterPlugin(FlashPlugin{})
	RegisterPlugin(ValidationPlugin{})
	RegisterPlugin(InterceptorPlugin{})
}
