package revel

import (
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

const GO_TEMPLATE = "go"

var ERROR_CLASS = "hasError"

// This object handles loading and parsing of templates.
// Everything below the application's views directory is treated as a template.
type TemplateLoader struct {
	// Template data and implementation
	templatesAndEngine TemplateEngine
	// If an error was encountered parsing the templates, it is stored here.
	compileError *Error
	// Paths to search for templates, in priority order.
	paths []string
	// Map from template name to the path from whence it was loaded.
	TemplatePaths map[string]string
}

var TemplateEngines = map[string]func(loader *TemplateLoader) (TemplateEngine, error){}

type TemplateEngine interface {
	// #ParseAndAdd: prase template string and add template to the set.
	//   arg: templateName string
	//   arg: templateSource *string
	//   arg: basePath string - where templates are located
	ParseAndAdd(templateName string, templateSource string, basePath string) *Error

	// #Lookup: returns Template corresponding to the given templateName
	//   arg: templateSet *abstractTemplateSet
	//   arg: templateName string
	Lookup(templateName string) Template
}

type TemplateWatcher interface {
	WatchDir(info os.FileInfo) bool
	WatchFile(basename string) bool
}

type Template interface {
	Name() string
	Content() []string
	Render(wr io.Writer, arg interface{}) error
}

var invalidSlugPattern = regexp.MustCompile(`[^a-z0-9 _-]`)
var whiteSpacePattern = regexp.MustCompile(`\s+`)

func NewTemplateLoader(paths []string) *TemplateLoader {
	loader := &TemplateLoader{
		paths: paths,
	}
	return loader
}

// Sets the template name from Config
// Sets the template API methods for parsing and storing templates before rendering
func (loader *TemplateLoader) createTemplateEngine() (TemplateEngine, error) {
	templateEngineName, _ := Config.String(REVEL_TEMPLATE_ENGINE)
	if "" == templateEngineName {
		templateEngineName = GO_TEMPLATE
	}
	factory := TemplateEngines[templateEngineName]
	if nil == factory {
		return nil, errors.New("Unknown template engine name - " + templateEngineName + ".")
	}
	templateEngine, err := factory(loader)
	if nil != err {
		return nil, errors.New("Failed to init template engine (" + templateEngineName + "), " + err.Error())
	}

	INFO.Println("init templates:", templateEngineName)
	return templateEngine, nil
}

// This scans the views directory and parses all templates as Go Templates.
// If a template fails to parse, the error is set on the loader.
// (It's awkward to refresh a single Go Template)
func (loader *TemplateLoader) Refresh() *Error {
	TRACE.Printf("Refreshing templates from %s", loader.paths)

	loader.compileError = nil
	loader.TemplatePaths = map[string]string{}

	// Walk through the template loader's paths and build up a template set.
	var templatesAndEngine, err = loader.createTemplateEngine()
	if nil != err {
		loader.compileError = &Error{
			Title:       "Panic (Template Loader)",
			Description: err.Error(),
		}
		return loader.compileError
	}

	var watcher, _ = templatesAndEngine.(TemplateWatcher)
	if nil == watcher {
		watcher = loader
	}

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

			// is it a symlinked template?
			link, err := os.Lstat(path)
			if err == nil && link.Mode()&os.ModeSymlink == os.ModeSymlink {
				TRACE.Println("symlink template:", path)
				// lookup the actual target & check for goodness
				targetPath, err := filepath.EvalSymlinks(path)
				if err != nil {
					ERROR.Println("Failed to read symlink", err)
					return err
				}
				targetInfo, err := os.Stat(targetPath)
				if err != nil {
					ERROR.Println("Failed to stat symlink target", err)
					return err
				}

				// set the template path to the target of the symlink
				path = targetPath
				info = targetInfo

				// need to save state and restore for recursive call to Walk on symlink
				tmp := fullSrcDir
				fullSrcDir = filepath.Dir(targetPath)
				filepath.Walk(targetPath, templateWalker)
				fullSrcDir = tmp
			}

			// Walk into watchable directories
			if info.IsDir() {
				if !watcher.WatchDir(info) {
					return filepath.SkipDir
				}
				return nil
			}

			// Only add watchable
			if !watcher.WatchFile(info.Name()) {
				return nil
			}

			var fileStr string

			// addTemplate loads a template file into the Go template loader so it can be rendered later
			addTemplate := func(templateName string) error {
				TRACE.Println("adding template: ", fullSrcDir, templateName)
				// Convert template names to use forward slashes, even on Windows.
				if os.PathSeparator == '\\' {
					templateName = strings.Replace(templateName, `\`, `/`, -1) // `
				}

				// If we already loaded a template of this name, skip it.
				if old := templatesAndEngine.Lookup(templateName); nil != old {
					WARN.Println("template is already exists: ", templateName, "\r\n\told file:",
						loader.TemplatePaths[templateName], "\r\n\tnew file:", path)
					return nil
				}

				loader.TemplatePaths[templateName] = path

				// Load the file
				fileBytes, err := ioutil.ReadFile(path)
				if err != nil {
					ERROR.Println("Failed reading file:", path)
					return nil
				}
				fileStr = string(fileBytes)

				// Add the next template and parse it
				if err := templatesAndEngine.ParseAndAdd(templateName, fileStr, basePath); nil != err {
					return err
				}
				return nil
			}

			templateName := filepath.ToSlash(path[len(fullSrcDir)+1:])
			err = addTemplate(templateName)

			// Store / report the first error encountered.
			if err != nil && loader.compileError == nil {
				loader.compileError, _ = err.(*Error)
				if nil == loader.compileError {
					_, line, description := parseTemplateError(err)
					loader.compileError = &Error{
						Title:       "Template Compilation Error",
						Path:        templateName,
						Description: description,
						Line:        line,
						SourceLines: strings.Split(fileStr, "\n"),
					}
				}
				ERROR.Printf("Template compilation error (In %s around line %d):\n\t%s",
					templateName, loader.compileError.Line, err.Error())
			} else if nil != err && strings.HasPrefix(templateName, "errors/") {
				compileError, _ := err.(*Error)
				if nil != compileError {
					ERROR.Printf("Template compilation error (In %s around line %d):\n\t%s",
						templateName, compileError.Line, err.Error())
				} else {
					ERROR.Printf("Template compilation error (In %s around line %d):\n\t%s",
						templateName, -1, err.Error())
				}
			}
			return nil
		}

		funcErr := filepath.Walk(fullSrcDir, templateWalker)

		// If there was an error with the Funcs, set it and return immediately.
		if funcErr != nil {
			loader.compileError = funcErr.(*Error)
			return loader.compileError
		}
	}

	// Note: compileError may or may not be set.
	loader.templatesAndEngine = templatesAndEngine
	return loader.compileError
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

// Return the Template with the given name.  The name is the template's path
// relative to a template loader root.
//
// An Error is returned if there was any problem with any of the templates.  (In
// this case, if a template is returned, it may still be usable.)
func (loader *TemplateLoader) Template(name string) (Template, error) {
	if loader.compileError != nil {
		return nil, loader.compileError
	}

	// Look up and return the template.
	tmpl := loader.templatesAndEngine.Lookup(name)

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

/////////////////////
// Template functions
/////////////////////

// Return a url capable of invoking a given controller method:
// "Application.ShowApp 123" => "/app/123"
func ReverseUrl(args ...interface{}) (template.URL, error) {
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

	return template.URL(MainRouter.Reverse(args[0].(string), argsByName).Url), nil
}

func Slug(text string) string {
	separator := "-"
	text = strings.ToLower(text)
	text = invalidSlugPattern.ReplaceAllString(text, "")
	text = whiteSpacePattern.ReplaceAllString(text, separator)
	text = strings.Trim(text, separator)
	return text
}
