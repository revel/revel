package revel

import (
	"html/template"
	"io"
	"log"
	"strings"
)

const GO_TEMPLATE = "go"

// Called on startup, initialized when the REVEL_BEFORE_MODULES_LOADED is called
func init() {
	AddInitEventHandler(func(typeOf Event, value interface{}) (responseOf EventResponse) {
		if typeOf == REVEL_BEFORE_MODULES_LOADED {
			RegisterTemplateLoader(GO_TEMPLATE, func(loader *TemplateLoader) (TemplateEngine, error) {
				// Set the template delimiters for the project if present, then split into left
				// and right delimiters around a space character

				TemplateDelims := Config.StringDefault("template.go.delimiters", "")
				var splitDelims []string
				if TemplateDelims != "" {
					splitDelims = strings.Split(TemplateDelims, " ")
					if len(splitDelims) != 2 {
						log.Fatalln("app.conf: Incorrect format for template.delimiters")
					}
				}

				return &GoEngine{
					loader:          loader,
					templateSet:     template.New("__root__").Funcs(TemplateFuncs),
					templatesByName: map[string]*GoTemplate{},
					splitDelims:     splitDelims,
				}, nil
			})
		}
		return
	})
}

// Adapter for Go Templates.
type GoTemplate struct {
	*template.Template
	engine *GoEngine
	*TemplateView
}

// return a 'revel.Template' from Go's template.
func (gotmpl GoTemplate) Render(wr io.Writer, arg interface{}) error {
	return gotmpl.Execute(wr, arg)
}

// The main template engine for Go
type GoEngine struct {
	// The template loader
	loader *TemplateLoader
	// THe current template set
	templateSet *template.Template
	// A map of templates by name
	templatesByName map[string]*GoTemplate
	// The delimiter that is used to indicate template code, defaults to {{
	splitDelims []string
	// True if map is case insensitive
	CaseInsensitive bool
}

// Convert the path to lower case if needed
func (i *GoEngine) ConvertPath(path string) string {
	if i.CaseInsensitive {
		return strings.ToLower(path)
	}
	return path
}

// Returns true if this engine can handle the response
func (i *GoEngine) Handles(templateView *TemplateView) bool {
	return EngineHandles(i, templateView)
}

// Parses the template vide and adds it to the template set
func (engine *GoEngine) ParseAndAdd(baseTemplate *TemplateView) error {
	// If alternate delimiters set for the project, change them for this set
	if engine.splitDelims != nil && strings.Index(baseTemplate.Location(), ViewsPath) > -1 {
		engine.templateSet.Delims(engine.splitDelims[0], engine.splitDelims[1])
	} else {
		// Reset to default otherwise
		engine.templateSet.Delims("", "")
	}
	templateSource := string(baseTemplate.FileBytes)
	templateName := engine.ConvertPath(baseTemplate.TemplateName)
	tpl, err := engine.templateSet.New(baseTemplate.TemplateName).Parse(templateSource)
	if nil != err {
		_, line, description := ParseTemplateError(err)
		return &Error{
			Title:       "Template Compilation Error",
			Path:        baseTemplate.TemplateName,
			Description: description,
			Line:        line,
			SourceLines: strings.Split(templateSource, "\n"),
		}
	}
	engine.templatesByName[templateName] = &GoTemplate{Template: tpl, engine: engine, TemplateView: baseTemplate}
	return nil
}

// Lookups the template name, to see if it is contained in this engine
func (engine *GoEngine) Lookup(templateName string) Template {
	// Case-insensitive matching of template file name
	if tpl, found := engine.templatesByName[engine.ConvertPath(templateName)]; found {
		return tpl
	}
	return nil
}

// Return the engine name
func (engine *GoEngine) Name() string {
	return GO_TEMPLATE
}

// An event listener to listen for Revel INIT events
func (engine *GoEngine) Event(action Event, i interface{}) {
	if action == TEMPLATE_REFRESH_REQUESTED {
		// At this point all the templates have been passed into the
		engine.templatesByName = map[string]*GoTemplate{}
		engine.templateSet = template.New("__root__").Funcs(TemplateFuncs)
		// Check to see what should be used for case sensitivity
		engine.CaseInsensitive = Config.BoolDefault("go.template.caseinsensitive", true)
	}
}
