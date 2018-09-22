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
	// prase template string and add template to the set.
	ParseAndAdd(basePath *TemplateView) error

	// returns Template corresponding to the given templateName, or nil
	Lookup(templateName string) Template

	// Fired by the template loader when events occur
	Event(event Event, arg interface{})

	// returns true if this engine should be used to parse the file specified in baseTemplate
	Handles(templateView *TemplateView) bool

	// returns the name of the engine
	Name() string
}

// The template view information
type TemplateView struct {
	TemplateName string // The name of the view
	FilePath     string // The file path (view relative)
	BasePath     string // The file system base path
	FileBytes    []byte // The file loaded
	EngineType   string // The name of the engine used to render the view
}

var templateLoaderMap = map[string]func(loader *TemplateLoader) (TemplateEngine, error){}

// Allow for templates to be registered during init but not initialized until application has been started
func RegisterTemplateLoader(key string, loader func(loader *TemplateLoader) (TemplateEngine, error)) (err error) {
	if _, found := templateLoaderMap[key]; found {
		err = fmt.Errorf("Template loader %s already exists", key)
	}
	templateLog.Debug("Registered template engine loaded", "name", key)
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

	templateLog.Debug("CreateTemplateEngine: init templates", "name", templateEngineName)
	return templateEngine, nil
}

// Passing in a comma delimited list of engine names to be used with this loader to parse the template files
func (loader *TemplateLoader) initializeEngines(runtimeLoader *templateRuntime, templateEngineNameList string) (err *Error) {
	// Walk through the template loader's paths and build up a template set.
	if templateEngineNameList == "" {
		templateEngineNameList = GO_TEMPLATE

	}
	runtimeLoader.templatesAndEngineList = []TemplateEngine{}
	for _, engine := range strings.Split(templateEngineNameList, ",") {
		engine := strings.TrimSpace(strings.ToLower(engine))

		if templateLoader, err := loader.CreateTemplateEngine(engine); err != nil {
			runtimeLoader.compileError = &Error{
				Title:       "Panic (Template Loader)",
				Description: err.Error(),
			}
			return runtimeLoader.compileError
		} else {
			// Always assign a default engine, switch it if it is specified in the config
			runtimeLoader.templatesAndEngineList = append(runtimeLoader.templatesAndEngineList, templateLoader)
		}
	}
	return
}

func EngineHandles(engine TemplateEngine, templateView *TemplateView) bool {
	if line, _, e := bufio.NewReader(bytes.NewBuffer(templateView.FileBytes)).ReadLine(); e == nil && string(line[:3]) == "#! " {
		// Extract the shebang and look at the rest of the line
		// #! pong2
		// #! go
		templateType := strings.TrimSpace(string(line[2:]))
		if engine.Name() == templateType {
			// Advance the read file bytes so it does not include the shebang
			templateView.FileBytes = templateView.FileBytes[len(line)+1:]
			templateView.EngineType = templateType
			return true
		}
	}
	filename := filepath.Base(templateView.FilePath)
	bits := strings.Split(filename, ".")
	if len(bits) > 2 {
		templateType := strings.TrimSpace(bits[len(bits)-2])
		if engine.Name() == templateType {
			templateView.EngineType = templateType
			return true
		}
	}
	return false
}
