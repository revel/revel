package revel

import (
	"path"
	"path/filepath"
	"strings"

	revConfig "github.com/revel/revel/config"
	robfigConfig "github.com/robfig/config"
)

var RevelModules = []string{"github.com/revel/modules/"}

func isRevelModules(slashPath string) bool {
	for _, module := range RevelModules {
		if strings.Contains(slashPath, module) {
			return true
		}
	}
	return false
}

type MixedEngine struct {
	loader                     *TemplateLoader
	templateSetsByPath         map[string]TemplateEngine
	templateSetsByTemplateName map[string]TemplateEngine
}

func (engine *MixedEngine) templateEngineNameFrom(templateName, basePath string) (string, *Error) {
	basePath = filepath.ToSlash(filepath.Join(strings.TrimSuffix(basePath, "/app/views"), "conf"))

	// Load app.conf from basePath
	config, err := revConfig.LoadContext("app.conf", []string{basePath})
	if err != nil || config == nil {
		return "", &Error{
			Title:       "Panic (Template Loader)",
			Description: "Failed to load template(" + templateName + "): cann't load " + basePath + "/app.conf:" + err.Error(),
		}
	}

	// Ensure that the selected runmode appears in app.conf.
	// If empty string is passed as the mode, treat it as "DEFAULT"
	mode := RunMode
	if mode == "" {
		mode = robfigConfig.DEFAULT_SECTION
	}
	if !config.HasSection(mode) {
		return "", &Error{
			Title:       "Panic (Template Loader)",
			Description: "Failed to load template(" + templateName + "): cann't load " + basePath + "/app.conf: No mode found: " + mode,
		}
	}
	config.SetSection(mode)

	templateEngineName := config.StringDefault(REVEL_TEMPLATE_ENGINE, "")
	// preventing recursive load template engine.
	if MIXED_TEMPLATE == templateEngineName {
		templateEngineName = ""
	}
	return templateEngineName, nil
}

func (engine *MixedEngine) ParseAndAdd(templateName string, templateSource string, basePath string) *Error {
	templateSet := engine.templateSetsByPath[basePath]
	if nil == templateSet {
		var templateEngineName string
		slashPath := filepath.ToSlash(basePath)
		if strings.HasSuffix(slashPath, "/app/views") {
			if isRevelModules(slashPath) {
				templateEngineName = GO_TEMPLATE
			} else {
				var err *Error
				templateEngineName, err = engine.templateEngineNameFrom(templateName, basePath)
				if nil != err {
					return err
				}
			}
		} else if slashPath == filepath.ToSlash(path.Join(RevelPath, "templates")) {
			templateEngineName = GO_TEMPLATE
		} else {
			return &Error{
				Title:       "Panic (Template Loader)",
				Description: "Failed to load template(" + templateName + "): cann't load " + slashPath + "/../../conf/app.conf: invalid views path.",
			}
		}
		var err error
		templateSet, err = engine.loader.CreateTemplateEngine(templateEngineName)
		if nil != err {
			return &Error{
				Title:       "Panic (Template Loader)",
				Description: "Failed to load template(" + templateName + "): " + err.Error(),
			}
		}
		engine.templateSetsByPath[basePath] = templateSet
	}

	err := templateSet.ParseAndAdd(templateName, templateSource, basePath)
	if nil != err {
		return err
	}
	engine.templateSetsByTemplateName[strings.ToLower(templateName)] = templateSet
	return nil
}

func (engine *MixedEngine) Lookup(templateName string) Template {
	templateSet := engine.templateSetsByTemplateName[strings.ToLower(templateName)]
	if nil == templateSet {
		return nil
	}
	return templateSet.Lookup(templateName)
}

func init() {
	TemplateEngines[MIXED_TEMPLATE] = func(loader *TemplateLoader) (TemplateEngine, error) {
		return &MixedEngine{
			loader:                     loader,
			templateSetsByPath:         make(map[string]TemplateEngine),
			templateSetsByTemplateName: make(map[string]TemplateEngine),
		}, nil
	}
}
