package revel

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var ERROR_CLASS = "hasError"

// This object handles loading and parsing of templates.
// Everything below the application's views directory is treated as a template.
type TemplateLoader struct {
  // Template data and implementation
  templatesAndEngine *templateAndEnvironment
	// If an error was encountered parsing the templates, it is stored here.
	compileError *Error
	// Paths to search for templates, in priority order.
	paths []string
	// Map from template name to the path from whence it was loaded.
	templatePaths map[string]string
}

var defaultTemplateEngineName string = "GoTemplate"
type templateAndEnvironment struct {
  // Default is set in defaultTemplateEngineName, can be changed in
  // app.conf via template.engine
  engineName string
	// This is the set of all templates under views
  templateSet *abstractTemplateSet
  // Chosen engine implementation of template API
  // #initialAddAndParse: initial prepare, add the first template
  // and parse it, returns splitDelims []string to
  // be used in #addAndParse
  //   arg: templateSet **abstractTemplateSet
  //   arg: templateName string
  //   arg: templateSource *string
  //   arg: basePath string - where templates are located
  // #addAndParse
  //   arg: templateSet **abstractTemplateSet
  //   arg: templateName string
  //   arg: templateSource *string
  //   arg: basePath string - where templates are located
  //   arg: splitDelims []string - some kind of delimiters,
  //        depend fully on the template engine, set in
  //        #initialAddAndParse
  // #lookup: returns *Template corresponding to the given templateName
  //   arg: templateSet *abstractTemplateSet
  //   arg: templateName string
  //   arg: loader *TemplateLoader - to be referenced in
  //        the adapter struct
  methods map[string]interface{}
}
type abstractTemplateSet interface{}

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

// This scans the views directory and parses all templates as Go Templates.
// If a template fails to parse, the error is set on the loader.
// (It's awkward to refresh a single Go Template)
func (loader *TemplateLoader) Refresh() *Error {
	TRACE.Printf("Refreshing templates from %s", loader.paths)

	loader.compileError = nil
	loader.templatePaths = map[string]string{}
  
  var splitDelims []string

	// Walk through the template loader's paths and build up a template set.
	var templatesAndEngine *templateAndEnvironment = nil
	for _, basePath := range loader.paths {
		// Walk only returns an error if the template loader is completely unusable
		// (namely, if one of the TemplateFuncs does not have an acceptable signature).
		funcErr := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
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

			var fileStr string

			// addTemplate allows the same template to be added multiple
			// times with different template names.
			addTemplate := func(templateName string) (err error) {
				// Convert template names to use forward slashes, even on Windows.
				if os.PathSeparator == '\\' {
					templateName = strings.Replace(templateName, `\`, `/`, -1) // `
				}

				// If we already loaded a template of this name, skip it.
				if _, ok := loader.templatePaths[templateName]; ok {
					return nil
				}
				loader.templatePaths[templateName] = path

				// Load the file if we haven't already
				if fileStr == "" {
					fileBytes, err := ioutil.ReadFile(path)
					if err != nil {
						ERROR.Println("Failed reading file:", path)
						return nil
					}

					fileStr = string(fileBytes)
				}

				if templatesAndEngine == nil {
					// Create the template set.  This panics if any of the funcs do not
					// conform to expectations or template engine is unknown, so we wrap it in a func and handle those
					// panics by serving an error page.
					var funcError *Error
					func() {
						defer func() {
							if err := recover(); err != nil {
								funcError = &Error{
									Title:       "Panic (Template Loader)",
									Description: fmt.Sprintln(err),
								}
							}
						}()

            // Setup, add the first template and parse it
            templatesAndEngine = new(templateAndEnvironment)
            templatesAndEngine.setupTemplateEngine()
            splitDelims, err = templatesAndEngine.methods["initialAddAndParse"].(
              func(templateSet **abstractTemplateSet, templateName string, templateSource *string, basePath string) (splitDelims []string, err error) )(
                &templatesAndEngine.templateSet, templateName, &fileStr, basePath)
          }()

          if funcError != nil {
						return funcError
					}

        } else {
          // Add the next template and parse it
          err = templatesAndEngine.methods["addAndParse"].(
            func(templateSet *abstractTemplateSet, templateName string, templateSource *string, basePath string, splitDelims []string) error)(
              templatesAndEngine.templateSet, templateName, &fileStr, basePath, splitDelims)
        }
        return err
      }

      templateName := path[len(basePath)+1:]

      // Lower case the file name for case-insensitive matching
      lowerCaseTemplateName := strings.ToLower(templateName)

      err = addTemplate(templateName)
			err = addTemplate(lowerCaseTemplateName)

			// Store / report the first error encountered.
			if err != nil && loader.compileError == nil {
				_, line, description := parseTemplateError(err)
				loader.compileError = &Error{
					Title:       "Template Compilation Error",
					Path:        templateName,
					Description: description,
					Line:        line,
					SourceLines: strings.Split(fileStr, "\n"),
				}
				ERROR.Printf("Template compilation error (In %s around line %d):\n%s",
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

// Sets the template name from Config
// Sets the template API methods for parsing and storing templates before rendering
func (templatesAndEngine *templateAndEnvironment) setupTemplateEngine() {
  templateEngineName, _ := Config.String("template.engine")
  templatesAndEngine.setTemplateEngineName(templateEngineName)
  templatesAndEngine.setTemplateEngineMethods()
}

// Stores the template name or defaultTemplateEngineName
func (templatesAndEngine *templateAndEnvironment) setTemplateEngineName(templateEngineName string) {
  if templateEngineName == "" {
    templateEngineName = defaultTemplateEngineName
  }
  templatesAndEngine.engineName = templateEngineName
}

// Sets the template API methods for parsing and storing templates before rendering
func (templatesAndEngine *templateAndEnvironment) setTemplateEngineMethods() {
  switch templatesAndEngine.engineName {
  case "HAML":
    templatesAndEngine.methods = TemplateAPIOfHAML
  case "GoTemplate":
    templatesAndEngine.methods = TemplateAPIOfGoTemplate
  default:
    panic("Unknown template engine name.")
  }
}

// Return the Template with the given name.  The name is the template's path
// relative to a template loader root.
//
// An Error is returned if there was any problem with any of the templates.  (In
// this case, if a template is returned, it may still be usable.)
func (loader *TemplateLoader) Template(name string) (Template, error) {
	// Lower case the file name to support case-insensitive matching
	name = strings.ToLower(name)
	// Look up and return the template.
  tmpl := loader.templatesAndEngine.methods["lookup"].(
    func(templateSet *abstractTemplateSet, templateName string, loader *TemplateLoader) *Template)(
      loader.templatesAndEngine.templateSet, name, loader)

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

  return *tmpl, err
}

/////////////////////
// Template functions
/////////////////////

// Return a url capable of invoking a given controller method:
// "Application.ShowApp 123" => "/app/123"
func ReverseUrl(args ...interface{}) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no arguments provided to reverse route")
	}

	action := args[0].(string)
	actionSplit := strings.Split(action, ".")
	if len(actionSplit) != 2 {
		return "", fmt.Errorf("reversing '%s', expected 'Controller.Action'", action)
	}

	// Look up the types.
	var c Controller
	if err := c.SetAction(actionSplit[0], actionSplit[1]); err != nil {
		return "", fmt.Errorf("reversing %s: %s", action, err)
	}

	// Unbind the arguments.
	argsByName := make(map[string]string)
	for i, argValue := range args[1:] {
		Unbind(argsByName, c.MethodType.Args[i].Name, argValue)
	}

	return MainRouter.Reverse(args[0].(string), argsByName).Url, nil
}

func Slug(text string) string {
	separator := "-"
	text = strings.ToLower(text)
	text = invalidSlugPattern.ReplaceAllString(text, "")
	text = whiteSpacePattern.ReplaceAllString(text, separator)
	text = strings.Trim(text, separator)
	return text
}
