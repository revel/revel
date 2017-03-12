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

var ErrorCSSClass = "hasError"

var default_template_engine_name = GO_TEMPLATE

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
}

var templateLoaderMap = map[string]func(loader *TemplateLoader) (TemplateEngine, error){}

type TemplateEngine interface {
	// #ParseAndAdd: prase template string and add template to the set.
	//   arg: templateName string
	//   arg: templateSource *string
	//   arg: basePath string - where templates are located
	ParseAndAdd(templateName string, templateSource []byte, basePath *BaseTemplate) error

	// #Lookup: returns Template corresponding to the given templateName
	//   arg: templateSet *abstractTemplateSet
	//   arg: templateName string
	Lookup(templateName string) Template
	Name() string
}

type TemplateWatcher interface {
	WatchDir(info os.FileInfo) bool
	WatchFile(basename string) bool
}

type Template interface {
	Name() string      // Name of template
	Content() []string // Content
	Render(wr io.Writer, arg interface{}) error
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
		templateEngineName = default_template_engine_name
	}
	factory := templateLoaderMap[templateEngineName]
	if nil == factory {
        fmt.Printf("registered factories %#v\n %s \n",templateLoaderMap,templateEngineName)
        panic("Run to here")
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
	if templateEngineNameList=="" {
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
			loader.templatesAndEngineList = append(loader.templatesAndEngineList,templateLoader)
		}
	}
    return
}

// Refresh method scans the views directory and parses all templates as Go Templates.
// If a template fails to parse, the error is set on the loader.
// (It's awkward to refresh a single Go Template)
func (loader *TemplateLoader) Refresh() (err *Error) {
	TRACE.Printf("Refreshing templates from %s", loader.paths)
    if len(loader.templatesAndEngineList)==0 {
        if 	err =loader.InitializeEngines(GO_TEMPLATE);err!=nil {
            return
        }
    }

	// Resort the paths, make sure the revel path is the last path,
	// so anything can override it
	revelTemplatePath := filepath.Join(RevelPath, "templates")
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
					_, line, description := parseTemplateError(err)
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
func (loader *TemplateLoader) findAndAddTemplate(path, fullSrcDir, basePath string) (fileBytes []byte,err error) {
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
	baseTemplate := &BaseTemplate{location: basePath}
	// Sniff the first line of the file, if it has a shebang read it and use it to determine the template
	if line, _, e := bufio.NewReader(bytes.NewBuffer(fileBytes)).ReadLine(); e == nil && string(line[:2]) == "#! " {
		// Advance the read file bytes so it does not include the shebang
		fileBytes = fileBytes[:len(line)]
		// Extract the shebang and look at the rest of the line
		// #! pong2
		// #! go
		templateType := strings.TrimSpace(string(line[2:]))
        for _,engine := range loader.templatesAndEngineList {
            if engine.Name()==templateType {
                _, err = loader.loadIntoEngine(engine, templateName, path, baseTemplate, fileBytes)
                return
            }
        }
        return fileBytes, fmt.Errorf("Template specified type %s but it is not loaded %s",templateType,path)
	}
    // Try all engines available
    var defaultError error
    for _, engine := range loader.templatesAndEngineList {
        if loaded, loaderr := loader.loadIntoEngine(engine, templateName, path, baseTemplate, fileBytes); loaded {
            TRACE.Printf("Engine '%s' compiled %s", engine.Name(), path)
            loader.TemplatePaths[templateName] = path
            return
        }  else {
            TRACE.Printf("Engine '%s' unable to compile %s %s", engine.Name(), path,loaderr)
            if defaultError==nil {
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

func (loader *TemplateLoader) loadIntoEngine(engine TemplateEngine, templateName, path string, baseTemplate *BaseTemplate, fileBytes []byte) (loaded bool, err error) {
	if template := engine.Lookup(templateName); template != nil {
		// Duplicate template found for engine
		TRACE.Println("template is already exists: ", templateName, "\r\n\told file:",
			loader.TemplatePaths[templateName], "\r\n\tnew file:", path)
		loaded = true
	}
	if err = engine.ParseAndAdd(templateName, fileBytes, baseTemplate); err == nil {
		loader.TemplatePaths[templateName] = path
		loaded = true
	} else {
		TRACE.Printf("Engine '%s' failed to compile %s %s", engine.Name(), path, err)
	}
	return
}
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
func parseTemplateError(err error) (templateName string, line int, description string) {
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

// Template returns the Template with the given name.  The name is the template's path
// relative to a template loader root.
//
// An Error is returned if there was any problem with any of the templates.  (In
// this case, if a template is returned, it may still be usable.)
func (loader *TemplateLoader) Template(name string) (tmpl Template, err error) {
	if loader.compileError != nil {
		return nil, loader.compileError
	}

	// Look up and return the template.
	for _, engine := range loader.templatesAndEngineList {
		if tmpl = engine.Lookup(name); tmpl != nil {
			break
		}
	}

	if tmpl == nil && err == nil {
		err = fmt.Errorf("Template %s not found.", name)
	}

	return
}

type BaseTemplate struct {
	location string
}

func (i *BaseTemplate) Location() string {
	return i.location
}

/////////////////////
// Template functions
/////////////////////

// ReverseURL returns a url capable of invoking a given controller method:
// "Application.ShowApp 123" => "/app/123"
func ReverseURL(args ...interface{}) (template.URL, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no arguments provided to reverse route")
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
