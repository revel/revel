package mvc

type Result interface {
	error
	Apply(req *Request, resp *Response)
}

// Action methods return this result to request a template be rendered.
type RenderTemplateResult struct {
	Controller *Controller
	ViewName string  // e.g. "Index"
	Arg interface{}  // e.g. a map[string]interface{}, or a struct
}

func (r *RenderTemplateResult) Apply(req *Request, resp *Response) {
	templateLoader := r.Controller.TemplateLoader

	// Refresh templates.
	err := templateLoader.LoadTemplates()
	if err != nil {
		c.Response.out.Write([]byte(err.Html()))
		return
	}

	// TODO: Put the session, request, flash, params, errors into context.

	// Render the template into the response buffer.
	err := templateLoader.RenderTemplate(c.Response.out, c.name + "/" + viewName + ".html", arg)
	if err != nil {
		c.Response.out.Write([]byte(err.Error()))
	}
}
