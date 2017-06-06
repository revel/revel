// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ErrorCSSClass httml CSS error class name
var ErrorCSSClass = "hasError"

// TemplateLoader object handles loading and parsing of templates.
// Everything below the application's views directory is treated as a template.
type TemplateLoader struct {
	// Template data and implementation
	templatesAndEngineList []TemplateEngine
	// If an error was encountered parsing the templates, it is stored here.
	compileError *Error
	// Paths to search for templates, in priority order.
	paths []string
	// Map from template name to the path from whence it was loaded.
	TemplatePaths map[string]string
	// A map of looked up template results
	TemplateMap   map[string]Template
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

func NewTemplateLoader(paths []string) *TemplateLoader {
	loader := &TemplateLoader{
		paths: paths,
	}
	return loader
}

// Refresh method scans the views directory and parses all templates as Go Templates.
// If a template fails to parse, the error is set on the loader.
// (It's awkward to refresh a single Go Template)
// Refresh method scans the views directory and parses all templates as Go Templates.
// If a template fails to parse, the error is set on the loader.
// (It's awkward to refresh a single Go Template)
func (loader *TemplateLoader) Refresh() (err *Error) {
	TRACE.Printf("Refreshing templates from %s", loader.paths)
	if len(loader.templatesAndEngineList) == 0 {
		if err = loader.InitializeEngines(GO_TEMPLATE); err != nil {
			return
		}
	}
	for _, engine := range loader.templatesAndEngineList {
		engine.Event(TEMPLATE_REFRESH_REQUESTED, nil)
	}
	fireEvent(TEMPLATE_REFRESH_REQUESTED, nil)
	defer func() {
		for _, engine := range loader.templatesAndEngineList {
			engine.Event(TEMPLATE_REFRESH_COMPLETED, nil)
		}
		fireEvent(TEMPLATE_REFRESH_COMPLETED, nil)
		// Reset the TemplateMap, we don't prepopulate the map because
		loader.TemplateMap = map[string]Template{}

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
	TRACE.Printf("Refreshing templates from %s", loader.paths)

	loader.compileError = nil
	loader.TemplatePaths = map[string]string{}

	for _, basePath := range loader.paths {
		// Walk only returns an error if the template loader is completely unusable
		// (namely, if one of the TemplateFuncs does not have an acceptable signature).

		// Handling symlinked directories
		var fullSrcDir string
		f, err := os.Lstat(basePath)
		if err == nil && f.Mode()&os.ModeSymlink == os.ModeSymlink {
			fullSrcDir, err = filepath.EvalSymlinks(basePath)
			if err != nil {
				panic(err)
			}
		} else {
			fullSrcDir = basePath
		}

		var templateWalker filepath.WalkFunc
		templateWalker = func(path string, info os.FileInfo, err error) error {
			if err != nil {
				ERROR.Println("error walking templates:", err)
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

			fileBytes, err := loader.findAndAddTemplate(path, fullSrcDir, basePath)

			// Store / report the first error encountered.
			if err != nil && loader.compileError == nil {
				loader.compileError, _ = err.(*Error)
				if nil == loader.compileError {
					_, line, description := ParseTemplateError(err)
					loader.compileError = &Error{
						Title:       "Template Compilation Error",
						Path:        path,
						Description: description,
						Line:        line,
						SourceLines: strings.Split(string(fileBytes), "\n"),
					}
				}
				ERROR.Printf("Template compilation error (In %s around line %d):\n\t%s",
					path, loader.compileError.Line, err.Error())
			} else if nil != err { //&& strings.HasPrefix(templateName, "errors/") {

				if compileError, ok := err.(*Error); ok {
					ERROR.Printf("Template compilation error (In %s around line %d):\n\t%s",
						path, compileError.Line, err.Error())
				} else {
					ERROR.Printf("Template compilation error (In %s ):\n\t%s",
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
			loader.compileError = funcErr.(*Error)
			return loader.compileError
		}
	}

	// Note: compileError may or may not be set.
	return loader.compileError
}

// WatchDir returns true of directory doesn't start with . (dot)
// otherwise false
func (loader *TemplateLoader) findAndAddTemplate(path, fullSrcDir, basePath string) (fileBytes []byte, err error) {
	templateName := filepath.ToSlash(path[len(fullSrcDir)+1:])
	// Convert template names to use forward slashes, even on Windows.
	if os.PathSeparator == '\\' {
		templateName = strings.Replace(templateName, `\`, `/`, -1) // `
	}

	// Check to see if template was found
	if place, found := loader.TemplatePaths[templateName]; found {
		TRACE.Println("Not Loading, template is already exists: ", templateName, "\r\n\told file:",
			place, "\r\n\tnew file:", path)
		return
	}

	fileBytes, err = ioutil.ReadFile(path)
	if err != nil {
		ERROR.Println("Failed reading file:", path)
		return
	}
	// if we have an engine picked for this template process it now
	baseTemplate := NewBaseTemplate(templateName, path, basePath, fileBytes)

	// Try to find a default engine for the file
	for _, engine := range loader.templatesAndEngineList {
		if engine.Handles(baseTemplate) {
			_, err = loader.loadIntoEngine(engine, baseTemplate)
			return
		}
	}

	// Try all engines available
	var defaultError error
	for _, engine := range loader.templatesAndEngineList {
		if loaded, loaderr := loader.loadIntoEngine(engine, baseTemplate); loaded {
			return
		} else {
			TRACE.Printf("Engine '%s' unable to compile %s %s", engine.Name(), path, loaderr)
			if defaultError == nil {
				defaultError = loaderr
			}
		}
	}

	// Assign the error from the first parser
	err = defaultError

	// No engines could be found return the err
	if err != nil {
		err = fmt.Errorf("Failed to parse template file using engines %s %s", path, err)
	}

	return
}

func (loader *TemplateLoader) loadIntoEngine(engine TemplateEngine, baseTemplate *TemplateView) (loaded bool, err error) {
	if loadedTemplate := engine.Lookup(baseTemplate.TemplateName); loadedTemplate != nil {
		// Duplicate template found for engine
		TRACE.Println("template already exists: ", baseTemplate.TemplateName, " in engine ", engine.Name(), "\r\n\told file:",
			loadedTemplate.Location(), "\r\n\tnew file:", baseTemplate.FilePath)
		loaded = true
		return
	}
	if err = engine.ParseAndAdd(baseTemplate); err == nil {
		loader.TemplatePaths[baseTemplate.TemplateName] = baseTemplate.FilePath
		TRACE.Printf("Engine '%s' compiled %s", engine.Name(), baseTemplate.FilePath)
		loaded = true
	} else {
		TRACE.Printf("Engine '%s' failed to compile %s %s", engine.Name(), baseTemplate.FilePath, err)
	}
	return
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
			ERROR.Println("Failed to parse line number from error message:", err)
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

// DEPRECATED Use TemplateLang, will be removed in future release
func (loader *TemplateLoader) Template(name string) (tmpl Template, err error) {
	return loader.TemplateLang(name, "")
}
// Template returns the Template with the given name.  The name is the template's path
// relative to a template loader root.
//
// An Error is returned if there was any problem with any of the templates.  (In
// this case, if a template is returned, it may still be usable.)
func (loader *TemplateLoader) TemplateLang(name, lang string) (tmpl Template, err error) {
	if loader.compileError != nil {
		return nil, loader.compileError
	}
// Attempt to load a localized template first.
	if lang != "" {
		// Look up and return the template.
		tmpl = loader.templateLoad(name + "." + lang)
	}
	// Return non localized version
	if tmpl == nil {
		tmpl = loader.templateLoad(name)
	}

	if tmpl == nil && err == nil {
		err = fmt.Errorf("Template %s not found.", name)
	}

	return
}
func (loader *TemplateLoader) templateLoad(name string) (tmpl Template) {
	if t,found := loader.TemplateMap[name];!found && t != nil {
		tmpl = t
	} else {
		// Look up and return the template.
		for _, engine := range loader.templatesAndEngineList {
			if tmpl = engine.Lookup(name); tmpl != nil {
				loader.TemplateMap[name] = tmpl
				break
			}
		}
	}
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
	return nil
}
func NewBaseTemplate(templateName, filePath, basePath string, fileBytes []byte) *TemplateView {
	return &TemplateView{TemplateName: templateName, FilePath: filePath, FileBytes: fileBytes, BasePath: basePath}
}

/////////////////////
// Template functions
/////////////////////

// ReverseURL returns a url capable of invoking a given controller method:
// "Application.ShowApp 123" => "/app/123"
func ReverseURL(args ...interface{}) (template.URL, error) {
	if len(args) == 0 {
		return "", errors.New("no arguments provided to reverse route")
	}

	action := args[0].(string)
	if action == "Root" {
		return template.URL(AppRoot), nil
	}
	actionSplit := strings.Split(action, ".")
	if len(actionSplit) != 2 {
		return "", fmt.Errorf("reversing '%s', expected 'Controller.Action'", action)
	}

	// Look up the types.
	var c Controller
	if err := c.SetAction(actionSplit[0], actionSplit[1]); err != nil {
		return "", fmt.Errorf("reversing %s: %s", action, err)
	}

	if len(c.MethodType.Args) < len(args)-1 {
		return "", fmt.Errorf("reversing %s: route defines %d args, but received %d",
			action, len(c.MethodType.Args), len(args)-1)
	}

	// Unbind the arguments.
	argsByName := make(map[string]string)
	for i, argValue := range args[1:] {
		Unbind(argsByName, c.MethodType.Args[i].Name, argValue)
	}

	return template.URL(MainRouter.Reverse(args[0].(string), argsByName).URL), nil
}

func Slug(text string) string {
	separator := "-"
	text = strings.ToLower(text)
	text = invalidSlugPattern.ReplaceAllString(text, "")
	text = whiteSpacePattern.ReplaceAllString(text, separator)
	text = strings.Trim(text, separator)
	return text
}
