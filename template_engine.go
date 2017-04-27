package revel

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type TemplateEngine interface {
	// #ParseAndAdd: prase template string and add template to the set.
	//   arg: basePath *BaseTemplate
	ParseAndAdd(basePath *BaseTemplate) error

	// #Lookup: returns Template corresponding to the given templateName
	//   arg: templateName string
	Lookup(templateName string) Template

	// #Event: Fired by the template loader when events occur
	//   arg: event string
	//   arg: arg interface{}
	Event(event int, arg interface{})

	// #IsEngineFor: returns true if this engine should be used to parse the file specified in baseTemplate
	//   arg: engine The calling engine
	//   arg: baseTemplate The base template
	IsEngineFor(engine TemplateEngine, baseTemplate *BaseTemplate) bool

	// #Name: Returns the name of the engine
	Name() string
}

const (
	TEMPLATE_REFRESH = iota
	TEMPLATE_REFRESH_COMPLETE
)

var templateLoaderMap = map[string]func(loader *TemplateLoader) (TemplateEngine, error){}

// Allow for templates to be registered during init but not initialized until application has been started
func RegisterTemplateLoader(key string, loader func(loader *TemplateLoader) (TemplateEngine, error)) (err error) {
	if _, found := templateLoaderMap[key]; found {
		err = fmt.Errorf("Template loader %s already exists", key)
	}
	templateLoaderMap[key] = loader
	return
}

// Sets the template name from Config
// Sets the template API methods for parsing and storing templates before rendering
func (loader *TemplateLoader) CreateTemplateEngine(templateEngineName string) (TemplateEngine, error) {
	if "" == templateEngineName {
		templateEngineName = GO_TEMPLATE
	}
	factory := templateLoaderMap[templateEngineName]
	if nil == factory {
		fmt.Printf("registered factories %#v\n %s \n", templateLoaderMap, templateEngineName)
		return nil, errors.New("Unknown template engine name - " + templateEngineName + ".")
	}
	templateEngine, err := factory(loader)
	if nil != err {
		return nil, errors.New("Failed to init template engine (" + templateEngineName + "), " + err.Error())
	}

	INFO.Println("init templates:", templateEngineName)
	return templateEngine, nil
}

// Passing in a comma delimited list of engine names to be used with this loader to parse the template files
func (loader *TemplateLoader) InitializeEngines(templateEngineNameList string) (err *Error) {
	// Walk through the template loader's paths and build up a template set.
	if templateEngineNameList == "" {
		templateEngineNameList = GO_TEMPLATE

	}
	loader.templatesAndEngineList = []TemplateEngine{}
	for _, engine := range strings.Split(templateEngineNameList, ",") {
		engine := strings.TrimSpace(strings.ToLower(engine))

		if templateLoader, err := loader.CreateTemplateEngine(engine); err != nil {
			loader.compileError = &Error{
				Title:       "Panic (Template Loader)",
				Description: err.Error(),
			}
			return loader.compileError
		} else {
			// Always assign a default engine, switch it if it is specified in the config
			loader.templatesAndEngineList = append(loader.templatesAndEngineList, templateLoader)
		}
	}
	return
}

// BaseTemplateEngine allows new engines to use the default
type BaseTemplateEngine struct {
	CaseInsensitiveMode bool
}

func (i *BaseTemplateEngine) IsEngineFor(engine TemplateEngine, description *BaseTemplate) bool {
	if line, _, e := bufio.NewReader(bytes.NewBuffer(description.FileBytes)).ReadLine(); e == nil && string(line[:3]) == "#! " {
		// Extract the shebang and look at the rest of the line
		// #! pong2
		// #! go
		templateType := strings.TrimSpace(string(line[2:]))
		if engine.Name() == templateType {
			// Advance the read file bytes so it does not include the shebang
			description.FileBytes = description.FileBytes[len(line)+1:]
			description.EngineType = templateType
			return true
		}
	}
	filename := filepath.Base(description.FilePath)
	bits := strings.Split(filename, ".")
	if len(bits) > 2 {
		templateType := strings.TrimSpace(bits[len(bits)-2])
		if engine.Name() == templateType {
			description.EngineType = templateType
			return true
		}
	}
	return false
}
func (i *BaseTemplateEngine) ConvertPath(path string) string {
	if i.CaseInsensitiveMode {
		return strings.ToLower(path)
	}
	return path
}
