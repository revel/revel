// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// ErrorCSSClass httml CSS error class name
var ErrorCSSClass = "hasError"

// TemplateLoader object handles loading and parsing of templates.
// Everything below the application's views directory is treated as a template.
type TemplateLoader struct {
	// Paths to search for templates, in priority order.
	paths []string
	// load version seed for templates
	loadVersionSeed int
	// A templateRuntime of looked up template results
	runtimeLoader atomic.Value
	// Lock to prevent concurrent map writes
	templateMutex sync.Mutex
}

type Template interface {
	// The name of the template.
	Name() string // Name of template
	// The content of the template as a string (Used in error handling).
	Content() []string // Content
	// Called by the server to render the template out the io.Writer, context contains the view args to be passed to the template.
	Render(wr io.Writer, context interface{}) error
	// The full path to the file on the disk.
	Location() string // Disk location
}

var invalidSlugPattern = regexp.MustCompile(`[^a-z0-9 _-]`)
var whiteSpacePattern = regexp.MustCompile(`\s+`)
var templateLog = RevelLog.New("section", "template")

// TemplateOutputArgs returns the result of the template rendered using the passed in arguments.
func TemplateOutputArgs(templatePath string, args map[string]interface{}) (data []byte, err error) {
	// Get the Template.
	lang, _ := args[CurrentLocaleViewArg].(string)
	template, err := MainTemplateLoader.TemplateLang(templatePath, lang)
	if err != nil {
		return nil, err
	}
	tr := &RenderTemplateResult{
		Template: template,
		ViewArgs: args,
	}
	b, err := tr.ToBytes()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func NewTemplateLoader(paths []string) *TemplateLoader {
	loader := &TemplateLoader{
		paths: paths,
	}
	return loader
}

// WatchDir returns true of directory doesn't start with . (dot)
// otherwise false
func (loader *TemplateLoader) WatchDir(info os.FileInfo) bool {
	// Watch all directories, except the ones starting with a dot.
	return !strings.HasPrefix(info.Name(), ".")
}

// WatchFile returns true of file doesn't start with . (dot)
// otherwise false
func (loader *TemplateLoader) WatchFile(basename string) bool {
	// Watch all files, except the ones starting with a dot.
	return !strings.HasPrefix(basename, ".")
}

// DEPRECATED Use TemplateLang, will be removed in future release
func (loader *TemplateLoader) Template(name string) (tmpl Template, err error) {
	runtimeLoader := loader.runtimeLoader.Load().(*templateRuntime)
	return runtimeLoader.TemplateLang(name, "")
}

func (loader *TemplateLoader) TemplateLang(name, lang string) (tmpl Template, err error) {
	runtimeLoader := loader.runtimeLoader.Load().(*templateRuntime)
	return runtimeLoader.TemplateLang(name, lang)
}

// Refresh method scans the views directory and parses all templates as Go Templates.
// If a template fails to parse, the error is set on the loader.
// (It's awkward to refresh a single Go Template)
func (loader *TemplateLoader) Refresh() (err *Error) {
	loader.templateMutex.Lock()
	defer loader.templateMutex.Unlock()

	loader.loadVersionSeed++
	runtimeLoader := &templateRuntime{loader: loader,
		version:     loader.loadVersionSeed,
		templateMap: map[string]Template{}}

	templateLog.Debug("Refresh: Refreshing templates from ", "path", loader.paths)
	if err = loader.initializeEngines(runtimeLoader, Config.StringDefault("template.engines", GO_TEMPLATE)); err != nil {
		return
	}
	for _, engine := range runtimeLoader.templatesAndEngineList {
		engine.Event(TEMPLATE_REFRESH_REQUESTED, nil)
	}
	RaiseEvent(TEMPLATE_REFRESH_REQUESTED, nil)
	defer func() {
		for _, engine := range runtimeLoader.templatesAndEngineList {
			engine.Event(TEMPLATE_REFRESH_COMPLETED, nil)
		}
		RaiseEvent(TEMPLATE_REFRESH_COMPLETED, nil)

		// Reset the runtimeLoader
		loader.runtimeLoader.Store(runtimeLoader)
	}()

	// Resort the paths, make sure the revel path is the last path,
	// so anything can override it
	revelTemplatePath := filepath.Join(RevelPath, "templates")
	// Go through the paths
	for i, o := range loader.paths {
		if o == revelTemplatePath && i != len(loader.paths)-1 {
			loader.paths[i] = loader.paths[len(loader.paths)-1]
			loader.paths[len(loader.paths)-1] = revelTemplatePath
		}
	}
	templateLog.Debug("Refresh: Refreshing templates from", "path", loader.paths)

	runtimeLoader.compileError = nil
	runtimeLoader.TemplatePaths = map[string]string{}

	for _, basePath := range loader.paths {
		// Walk only returns an error if the template loader is completely unusable
		// (namely, if one of the TemplateFuncs does not have an acceptable signature).

		// Handling symlinked directories
		var fullSrcDir string
		f, err := os.Lstat(basePath)
		if err == nil && f.Mode()&os.ModeSymlink == os.ModeSymlink {
			fullSrcDir, err = filepath.EvalSymlinks(basePath)
			if err != nil {
				templateLog.Panic("Refresh: Eval symlinks error ", "error", err)
			}
		} else {
			fullSrcDir = basePath
		}

		var templateWalker filepath.WalkFunc

		templateWalker = func(path string, info os.FileInfo, err error) error {
			if err != nil {
				templateLog.Error("Refresh: error walking templates:", "error", err)
				return nil
			}

			// Walk into watchable directories
			if info.IsDir() {
				if !loader.WatchDir(info) {
					return filepath.SkipDir
				}
				return nil
			}

			// Only add watchable
			if !loader.WatchFile(info.Name()) {
				return nil
			}

			fileBytes, err := runtimeLoader.findAndAddTemplate(path, fullSrcDir, basePath)
			if err != nil {
				// Add in this template name to the list of templates unable to be compiled
				runtimeLoader.compileErrorNameList = append(runtimeLoader.compileErrorNameList, filepath.ToSlash(path[len(fullSrcDir)+1:]))
			}
			// Store / report the first error encountered.
			if err != nil && runtimeLoader.compileError == nil {
				runtimeLoader.compileError, _ = err.(*Error)

				if nil == runtimeLoader.compileError {
					_, line, description := ParseTemplateError(err)

					runtimeLoader.compileError = &Error{
						Title:       "Template Compilation Error",
						Path:        path,
						Description: description,
						Line:        line,
						SourceLines: strings.Split(string(fileBytes), "\n"),
					}
				}
				templateLog.Errorf("Refresh: Template compilation error (In %s around line %d):\n\t%s",
					path, runtimeLoader.compileError.Line, err.Error())
			} else if nil != err { //&& strings.HasPrefix(templateName, "errors/") {

				if compileError, ok := err.(*Error); ok {
					templateLog.Errorf("Template compilation error (In %s around line %d):\n\t%s",
						path, compileError.Line, err.Error())
				} else {
					templateLog.Errorf("Template compilation error (In %s ):\n\t%s",
						path, err.Error())
				}
			}
			return nil
		}

		if _, err = os.Lstat(fullSrcDir); os.IsNotExist(err) {
			// #1058 Given views/template path is not exists
			// so no need to walk, move on to next path
			continue
		}

		funcErr := Walk(fullSrcDir, templateWalker)

		// If there was an error with the Funcs, set it and return immediately.
		if funcErr != nil {
			runtimeLoader.compileError = NewErrorFromPanic(funcErr)
			return runtimeLoader.compileError
		}
	}

	// Note: compileError may or may not be set.
	return runtimeLoader.compileError
}

type templateRuntime struct {
	loader *TemplateLoader
	// load version for templates
	version int
	// Template data and implementation
	templatesAndEngineList []TemplateEngine
	// If an error was encountered parsing the templates, it is stored here.
	compileError *Error
	// A list of the names of the templates with errors
	compileErrorNameList []string
	// Map from template name to the path from whence it was loaded.
	TemplatePaths map[string]string
	// A map of looked up template results
	templateMap map[string]Template
}

// Checks to see if template exists in templatePaths, if so it is skipped (templates are imported in order
// reads the template file into memory, replaces namespace keys with module (if found
func (runtimeLoader *templateRuntime) findAndAddTemplate(path, fullSrcDir, basePath string) (fileBytes []byte, err error) {
	templateName := filepath.ToSlash(path[len(fullSrcDir)+1:])
	// Convert template names to use forward slashes, even on Windows.
	if os.PathSeparator == '\\' {
		templateName = strings.Replace(templateName, `\`, `/`, -1) // `
	}

	// Check to see if template was found
	if place, found := runtimeLoader.TemplatePaths[templateName]; found {
		templateLog.Debug("findAndAddTemplate: Not Loading, template is already exists: ", "name", templateName, "old",
			place, "new", path)
		return
	}

	fileBytes, err = ioutil.ReadFile(path)
	if err != nil {
		templateLog.Error("findAndAddTemplate: Failed reading file:", "path", path, "error", err)
		return
	}
	// Parse template file and replace the "_LOCAL_|" in the template with the module name
	// allow for namespaces to be renamed "_LOCAL_(.*?)|"
	if module := ModuleFromPath(path, false); module != nil {
		fileBytes = namespaceReplace(fileBytes, module)
	}

	// if we have an engine picked for this template process it now
	baseTemplate := NewBaseTemplate(templateName, path, basePath, fileBytes)

	// Try to find a default engine for the file
	for _, engine := range runtimeLoader.templatesAndEngineList {
		if engine.Handles(baseTemplate) {
			_, err = runtimeLoader.loadIntoEngine(engine, baseTemplate)
			return
		}
	}

	// Try all engines available
	var defaultError error
	for _, engine := range runtimeLoader.templatesAndEngineList {
		if loaded, loaderr := runtimeLoader.loadIntoEngine(engine, baseTemplate); loaded {
			return
		} else {
			templateLog.Debugf("findAndAddTemplate: Engine '%s' unable to compile %s %s", engine.Name(), path, loaderr.Error())
			if defaultError == nil {
				defaultError = loaderr
			}
		}
	}

	// Assign the error from the first parser
	err = defaultError

	// No engines could be found return the err
	if err == nil {
		err = fmt.Errorf("Failed to parse template file using engines %s", path)
	}

	return
}

func (runtimeLoader *templateRuntime) loadIntoEngine(engine TemplateEngine, baseTemplate *TemplateView) (loaded bool, err error) {
	if loadedTemplate, found := runtimeLoader.templateMap[baseTemplate.TemplateName]; found {
		// Duplicate template found in map
		templateLog.Debug("template already exists in map: ", baseTemplate.TemplateName, " in engine ", engine.Name(), "\r\n\told file:",
			loadedTemplate.Location(), "\r\n\tnew file:", baseTemplate.FilePath)
		return
	}

	if loadedTemplate := engine.Lookup(baseTemplate.TemplateName); loadedTemplate != nil {
		// Duplicate template found for engine
		templateLog.Debug("loadIntoEngine: template already exists: ", "template", baseTemplate.TemplateName, "inengine ", engine.Name(), "old",
			loadedTemplate.Location(), "new", baseTemplate.FilePath)
		loaded = true
		return
	}
	if err = engine.ParseAndAdd(baseTemplate); err == nil {
		if tmpl := engine.Lookup(baseTemplate.TemplateName); tmpl != nil {
			runtimeLoader.templateMap[baseTemplate.TemplateName] = tmpl
		}
		runtimeLoader.TemplatePaths[baseTemplate.TemplateName] = baseTemplate.FilePath
		templateLog.Debugf("loadIntoEngine:Engine '%s' compiled %s", engine.Name(), baseTemplate.FilePath)
		loaded = true
	} else {
		templateLog.Debug("loadIntoEngine: Engine failed to compile", "engine", engine.Name(), "file", baseTemplate.FilePath, "error", err)
	}
	return
}

// Parse the line, and description from an error message like:
// html/template:Application/Register.html:36: no such template "footer.html"
func ParseTemplateError(err error) (templateName string, line int, description string) {
	if e, ok := err.(*Error); ok {
		return "", e.Line, e.Description
	}

	description = err.Error()
	i := regexp.MustCompile(`:\d+:`).FindStringIndex(description)
	if i != nil {
		line, err = strconv.Atoi(description[i[0]+1 : i[1]-1])
		if err != nil {
			templateLog.Error("ParseTemplateError: Failed to parse line number from error message:", "error", err)
		}
		templateName = description[:i[0]]
		if colon := strings.Index(templateName, ":"); colon != -1 {
			templateName = templateName[colon+1:]
		}
		templateName = strings.TrimSpace(templateName)
		description = description[i[1]+1:]
	}
	return templateName, line, description
}

// Template returns the Template with the given name.  The name is the template's path
// relative to a template loader root.
//
// An Error is returned if there was any problem with any of the templates.  (In
// this case, if a template is returned, it may still be usable.)
func (runtimeLoader *templateRuntime) TemplateLang(name, lang string) (tmpl Template, err error) {
	if runtimeLoader.compileError != nil {
		for _, errName := range runtimeLoader.compileErrorNameList {
			if name == errName {
				return nil, runtimeLoader.compileError
			}
		}
	}

	// Fetch the template from the map
	tmpl = runtimeLoader.templateLoad(name, lang)
	if tmpl == nil {
		err = fmt.Errorf("Template %s not found.", name)
	}

	return
}

// Load and also updates map if name is not found (to speed up next lookup)
func (runtimeLoader *templateRuntime) templateLoad(name, lang string) (tmpl Template) {
	langName := name
	found := false
	if lang != "" {
		// Look up and return the template.
		langName = name + "." + lang
		tmpl, found = runtimeLoader.templateMap[langName]
		if found {
			return
		}
		tmpl, found = runtimeLoader.templateMap[name]
	} else {
		tmpl, found = runtimeLoader.templateMap[name]
		if found {
			return
		}
	}

	if !found {
		// Neither name is found
		// Look up and return the template.
		for _, engine := range runtimeLoader.templatesAndEngineList {
			if tmpl = engine.Lookup(langName); tmpl != nil {
				found = true
				break
			}
			if tmpl = engine.Lookup(name); tmpl != nil {
				found = true
				break
			}
		}
		if !found {
			return
		}
	}

	// If we found anything store it in the map, we need to copy so we do not
	// run into concurrency issues
	runtimeLoader.loader.templateMutex.Lock()
	defer runtimeLoader.loader.templateMutex.Unlock()

	// In case another thread has loaded the map, reload the atomic value and check
	newRuntimeLoader := runtimeLoader.loader.runtimeLoader.Load().(*templateRuntime)
	if newRuntimeLoader.version != runtimeLoader.version {
		return
	}

	newTemplateMap := map[string]Template{}
	for k, v := range newRuntimeLoader.templateMap {
		newTemplateMap[k] = v
	}
	newTemplateMap[langName] = tmpl
	if _, found := newTemplateMap[name]; !found {
		newTemplateMap[name] = tmpl
	}
	runtimeCopy := &templateRuntime{}
	*runtimeCopy = *newRuntimeLoader
	runtimeCopy.templateMap = newTemplateMap

	// Set the atomic value
	runtimeLoader.loader.runtimeLoader.Store(runtimeCopy)
	return
}

func (i *TemplateView) Location() string {
	return i.FilePath
}

func (i *TemplateView) Content() (content []string) {
	if i.FileBytes != nil {
		// Parse the bytes
		buffer := bytes.NewBuffer(i.FileBytes)
		reader := bufio.NewScanner(buffer)
		for reader.Scan() {
			content = append(content, string(reader.Bytes()))
		}
	}

	return content
}

func NewBaseTemplate(templateName, filePath, basePath string, fileBytes []byte) *TemplateView {
	return &TemplateView{TemplateName: templateName, FilePath: filePath, FileBytes: fileBytes, BasePath: basePath}
}
