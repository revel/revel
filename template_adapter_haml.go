package revel

import (
  "github.com/realistschuckle/gohaml"
  "io"
)

var TemplateAPIOfHAML = map[string]interface{}{
  "initialAddAndParse": func(templateSet **abstractTemplateSet, templateName string, templateSource *string, basePath string) (splitDelims []string, err error) {
    /*var scope = make(map[string]interface{})
    scope["lang"] = "HAML"
    content := "I love <\n=lang<\n!"
    output := engine.Render(scope)
    */
    engine, _ := gohaml.NewEngine(*templateSource)
    var hamlTemplateSet abstractTemplateSet = map[string]*gohaml.Engine{templateName: engine}
    *templateSet = &hamlTemplateSet
    return
  },
  "addAndParse": func(templateSet *abstractTemplateSet, templateName string, templateSource *string, basePath string, splitDelims []string) error {
    //HAMLTemplateSet := HAMLTemplate(*templateSet)
    return nil
  },
  "lookup": func(templateSet *abstractTemplateSet, templateName string, loader *TemplateLoader) *Template {
    //return HAMLTemplate{tmpl, loader}, err
    //HAMLTemplateSet := HAMLTemplate(*templateSet)
    return nil
  },
}

// Adapter for HAML Templates.
type HAMLTemplate struct {
	//*gohaml.Template
  template interface{}
	loader *TemplateLoader
}

func (haml HAMLTemplate) Name() string {
  return "my haml name"
}

// return a 'revel.Template' from Go's template.
func (haml HAMLTemplate) Render(wr io.Writer, arg interface{}) error {
  return nil
}

func (haml HAMLTemplate) Content() []string {
	content, _ := ReadLines(haml.loader.templatePaths[haml.Name()])
	return content
}
