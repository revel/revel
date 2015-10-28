package revel

import (
	"path/filepath"
	"strings"

	revConfig "github.com/revel/revel/config"
	robfigConfig "github.com/robfig/config"
)

type MixedEngine struct {
	loader                     *TemplateLoader
	templateSetsByPath         map[string]TemplateEngine
	templateSetsByTemplateName map[string]TemplateEngine
}

func (engine *MixedEngine) ParseAndAdd(templateName string, templateSource string, basePath string) *Error {
	templateSet := engine.templateSetsByPath[basePath]
	if nil == templateSet {
		confPath := filepath.ToSlash(basePath)
		if strings.HasSuffix(confPath, "/app/views") {
			confPath = filepath.ToSlash(filepath.Join(strings.TrimSuffix(confPath, "/app/views"), "conf"))
		}

		// Load app.conf from confPath
		config, err := revConfig.LoadContext("app.conf", []string{confPath})
		if err != nil || config == nil {
			return &Error{
				Title:       "Panic (Template Loader)",
				Description: "Failed to load template(" + templateName + "): cann't load " + confPath + "/app.conf:" + err.Error(),
			}
		}

		// Ensure that the selected runmode appears in app.conf.
		// If empty string is passed as the mode, treat it as "DEFAULT"
		mode := RunMode
		if mode == "" {
			mode = robfigConfig.DEFAULT_SECTION
		}
		if !config.HasSection(mode) {
			return &Error{
				Title:       "Panic (Template Loader)",
				Description: "Failed to load template(" + templateName + "): cann't load " + confPath + "/app.conf: No mode found: " + mode,
			}
		}
		config.SetSection(mode)

		templateEngineName := config.StringDefault(REVEL_TEMPLATE_ENGINE, "")
		// preventing recursive load template engine.
		if MIXED_TEMPLATE == templateEngineName {
			templateEngineName = ""
		}
		templateSet, err = engine.loader.CreateTemplateEngine(templateEngineName)
		if nil != err {
			return &Error{
				Title:       "Panic (Template Loader)",
				Description: "Failed to load template(" + templateName + "): " + err.Error(),
			}
		}
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
