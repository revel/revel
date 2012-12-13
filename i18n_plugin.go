package rev

func init() {
	RegisterPlugin(I18nPlugin{})
}

type I18nPlugin struct {
	EmptyPlugin
}

func (p I18nPlugin) BeforeRequest(c *Controller) {
	if c.Request.AcceptLanguages != nil {
		TRACE.Printf("Using accepted languages: %s", c.Request.AcceptLanguages)
	}
}
