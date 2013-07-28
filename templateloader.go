package revel

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

// TemplateLoader handles loading of templates, by passing them to the
// appropriate TemplateEngine.
type TemplateLoader struct {
	// If an error was encountered parsing the templates, it is stored here.
	compileError *Error
	// Paths to search for templates, in priority order.
	paths []string
	// Map from template name to the path from whence it was loaded.
	templatePaths map[string]string
	// Map from file extension to the template engine that should handle it.
	engines map[string]TemplateEngine
}

func NewTemplateLoader(paths []string) *TemplateLoader {
	return &TemplateLoader{
		paths: paths,
	}
}

// Refresh scans the views directory and parses all templates using the
// configured TemplateEngines.  If a template fails to parse, the error is set
// on the loader (and returned).
func (loader *TemplateLoader) Refresh() *Error {
	glog.V(1).Infof("Refreshing templates from %s", loader.paths)
	loader.compileError = nil
	loader.templatePaths = map[string]string{}

	// Set the template delimiters for the project if present, then split into left
	// and right delimiters around a space character
	var splitDelims []string
	if delims := Config.StringDefault("template.delimiters", ""); delims != "" {
		splitDelims = strings.Split(delims, " ")
		if len(splitDelims) != 2 {
			glog.Fatalln("app.conf: Incorrect format for template.delimiters")
		}
	}

	loader.engines = map[string]TemplateEngine{
		".html": NewHtmlTemplateEngine(),
		".xml":  NewHtmlTemplateEngine(),
		".json": NewTextTemplateEngine(),
		".txt":  NewTextTemplateEngine(),
	}

	// Walk through the template loader's paths and pass each template to the
	// appropriate engine.
	for _, basePath := range loader.paths {

		// Walk only returns an error if the template loader is completely unusable
		// (namely, if one of the TemplateFuncs does not have an acceptable signature).
		funcErr := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) (walkErr error) {
			defer func() {
				if err := recover(); err != nil {
					walkErr = &Error{
						Title:       "Panic (Template Loader)",
						Description: fmt.Sprintln(err),
					}
				}
			}()

			if err != nil {
				glog.Errorln("error walking templates:", err)
				return nil
			}

			// Walk into directories.
			if info.IsDir() {
				if !loader.WatchDir(info) {
					return filepath.SkipDir
				}
				return nil
			}

			if !loader.WatchFile(info.Name()) {
				return nil
			}

			// Convert template names to use forward slashes, even on Windows.
			templateName := path[len(basePath)+1:]
			if os.PathSeparator == '\\' {
				templateName = strings.Replace(templateName, "\\", "/", -1)
			}

			// If we already loaded a template of this name, skip it.
			if _, ok := loader.templatePaths[templateName]; ok {
				return nil
			}
			loader.templatePaths[templateName] = path

			fileBytes, err := ioutil.ReadFile(path)
			if err != nil {
				glog.Errorln("Failed reading file:", path)
				return nil
			}

			ext := filepath.Ext(templateName)
			engine, ok := loader.engines[ext]
			if !ok {
				glog.Warningln("No template engine for file:", templateName)
				return nil
			}

			// If alternate delimiters set for the project, change them for this template.
			if splitDelims != nil {
				if strings.HasPrefix(path, ViewsPath) {
					engine.Delims(splitDelims[0], splitDelims[1])
				} else {
					engine.Delims("", "")
				}
			}

			err = engine.Parse(templateName, string(fileBytes))

			// Store / report the first error encountered.
			if err != nil && loader.compileError == nil {
				_, line, description := parseTemplateError(err)
				loader.compileError = &Error{
					Title:       "Template Compilation Error",
					Path:        templateName,
					Description: description,
					Line:        line,
					SourceLines: strings.Split(string(fileBytes), "\n"),
				}
				glog.Errorf("Template compilation error (In %s around line %d):\n%s",
					templateName, line, description)
			}

			return nil
		})

		// If there was an error with the Funcs, set it and return immediately.
		if funcErr != nil {
			loader.compileError = funcErr.(*Error)
			return loader.compileError
		}
	}
	return loader.compileError
}

// SourceLines returns the template's source code.
// A template of the given name must exist, or a panic results.
func (loader *TemplateLoader) SourceLines(templateName string) []string {
	path, ok := loader.templatePaths[templateName]
	if !ok {
		panic("template not found: " + templateName)
	}

	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		glog.Errorln("failed reading file", path, ":", err)
		return []string{}
	}

	// TODO: Better way to split into lines?
	return strings.Split(string(fileBytes), "\n")
}

func (loader *TemplateLoader) WatchDir(info os.FileInfo) bool {
	// Watch all directories, except the ones starting with a dot.
	return !strings.HasPrefix(info.Name(), ".")
}

func (loader *TemplateLoader) WatchFile(basename string) bool {
	// Watch all files, except the ones starting with a dot.
	return !strings.HasPrefix(basename, ".")
}

// Parse the line, and description from an error message like:
// html/template:Application/Register.html:36: no such template "footer.html"
func parseTemplateError(err error) (templateName string, line int, description string) {
	description = err.Error()
	i := regexp.MustCompile(`:\d+:`).FindStringIndex(description)
	if i != nil {
		line, err = strconv.Atoi(description[i[0]+1 : i[1]-1])
		if err != nil {
			glog.Errorln("Failed to parse line number from error message:", err)
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

// Return the Template with the given name.  The name is the template's path
// relative to a template loader root.
//
// An Error is returned if there was any problem with any of the templates.  (In
// this case, if a template is returned, it may still be usable.)
func (loader *TemplateLoader) Template(name string) (Template, error) {
	// Figure out which template engine to query.
	ext := filepath.Ext(name)
	engine, ok := loader.engines[ext]
	if !ok {
		return nil, fmt.Errorf("error load %s: engine not found for extension %s", name, ext)
	}

	// Look up and return the template.
	tmpl := engine.Lookup(name)

	// This is necessary.
	// If a nil loader.compileError is returned directly, a caller testing against
	// nil will get the wrong result.  Something to do with casting *Error to error.
	var err error
	if loader.compileError != nil {
		err = loader.compileError
	}

	if tmpl == nil && err == nil {
		return nil, fmt.Errorf("Template %s not found.", name)
	}

	return tmpl, err
}
