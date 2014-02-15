package revel

import (
  "github.com/realistschuckle/gohaml"
  "io"
)

var TemplateAPIOfHAML = map[string]interface{}{
  "initialAddAndParse": func(templateSet **abstractTemplateSet, templateName string, templateSource *string, basePath string) (splitDelims []string, err error) {
    if engine, err := gohaml.NewEngine(*templateSource); err == nil {
      hamlTemplateSet := make(HAMLTemplateSet)
      hamlTemplateSet[templateName] = HAMLTemplate{templateName, engine, nil}
      var abstractTemplateSet abstractTemplateSet = hamlTemplateSet
      *templateSet = &abstractTemplateSet
    }
    return
  },
  "addAndParse": func(templateSet *abstractTemplateSet, templateName string, templateSource *string, basePath string, splitDelims []string) (err error) {
    if engine, err := gohaml.NewEngine(*templateSource); err == nil {
      (*templateSet).(HAMLTemplateSet)[templateName] = HAMLTemplate{templateName, engine, nil}
    }
    return
  },
  "lookup": func(templateSet *abstractTemplateSet, templateName string, loader *TemplateLoader) *Template {
    var tmpl Template = (*templateSet).(HAMLTemplateSet)[templateName]
    return &tmpl
  },
}

// Adapter for HAML Templates.
type HAMLTemplate struct {
  name string
  template *gohaml.Engine
	loader *TemplateLoader
}
type HAMLTemplateSet map[string]HAMLTemplate

func (haml HAMLTemplate) Name() string {
  return haml.name
}

// return a 'revel.Template' from HAML's template.
func (haml HAMLTemplate) Render(wr io.Writer, arg interface{}) (err error) {
  _, err = io.WriteString(wr, haml.template.Render(arg.(map[string]interface{})))
  return
}

func (haml HAMLTemplate) Content() []string {
	content, _ := ReadLines(haml.loader.templatePaths[haml.Name()])
	return content
}
