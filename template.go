package revel

import (
	"fmt"
  "github.com/realistschuckle/gohaml"
	"html"
  "html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
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

type templateAndEnvironment struct {
  // Default "GoTemplate", can be changed in app.conf via template.engine
  engineName string
	// This is the set of all templates under views
  templateSet *abstractTemplateSet
  // Chosen engine implementation of template API
  methods map[string]interface{}
}
type abstractTemplateSet interface{}

type HAMLTemplateSet map[string]*gohaml.Engine
type GoTemplateSet template.Template

type Template interface {
	Name() string
	Content() []string
	Render(wr io.Writer, arg interface{}) error
}

var invalidSlugPattern = regexp.MustCompile(`[^a-z0-9 _-]`)
var whiteSpacePattern = regexp.MustCompile(`\s+`)

var (
	// The functions available for use in the templates.
	TemplateFuncs = map[string]interface{}{
		"url": ReverseUrl,
		"eq":  Equal,
		"set": func(renderArgs map[string]interface{}, key string, value interface{}) template.HTML {
			renderArgs[key] = value
			return template.HTML("")
		},
		"append": func(renderArgs map[string]interface{}, key string, value interface{}) template.HTML {
			if renderArgs[key] == nil {
				renderArgs[key] = []interface{}{value}
			} else {
				renderArgs[key] = append(renderArgs[key].([]interface{}), value)
			}
			return template.HTML("")
		},
		"field": NewField,
		"option": func(f *Field, val, label string) template.HTML {
			selected := ""
			if f.Flash() == val {
				selected = " selected"
			}
			return template.HTML(fmt.Sprintf(`<option value="%s"%s>%s</option>`,
				html.EscapeString(val), selected, html.EscapeString(label)))
		},
		"radio": func(f *Field, val string) template.HTML {
			checked := ""
			if f.Flash() == val {
				checked = " checked"
			}
			return template.HTML(fmt.Sprintf(`<input type="radio" name="%s" value="%s"%s>`,
				html.EscapeString(f.Name), html.EscapeString(val), checked))
		},
		"checkbox": func(f *Field, val string) template.HTML {
			checked := ""
			if f.Flash() == val {
				checked = " checked"
			}
			return template.HTML(fmt.Sprintf(`<input type="checkbox" name="%s" value="%s"%s>`,
				html.EscapeString(f.Name), html.EscapeString(val), checked))
		},
		// Pads the given string with &nbsp;'s up to the given width.
		"pad": func(str string, width int) template.HTML {
			if len(str) >= width {
				return template.HTML(html.EscapeString(str))
			}
			return template.HTML(html.EscapeString(str) + strings.Repeat("&nbsp;", width-len(str)))
		},

		"errorClass": func(name string, renderArgs map[string]interface{}) template.HTML {
			errorMap, ok := renderArgs["errors"].(map[string]*ValidationError)
			if !ok || errorMap == nil {
				WARN.Println("Called 'errorClass' without 'errors' in the render args.")
				return template.HTML("")
			}
			valError, ok := errorMap[name]
			if !ok || valError == nil {
				return template.HTML("")
			}
			return template.HTML(ERROR_CLASS)
		},

		"msg": func(renderArgs map[string]interface{}, message string, args ...interface{}) template.HTML {
			return template.HTML(Message(renderArgs[CurrentLocaleRenderArg].(string), message, args...))
		},

		// Replaces newlines with <br>
		"nl2br": func(text string) template.HTML {
			return template.HTML(strings.Replace(template.HTMLEscapeString(text), "\n", "<br>", -1))
		},

		// Skips sanitation on the parameter.  Do not use with dynamic data.
		"raw": func(text string) template.HTML {
			return template.HTML(text)
		},

		// Pluralize, a helper for pluralizing words to correspond to data of dynamic length.
		// items - a slice of items, or an integer indicating how many items there are.
		// pluralOverrides - optional arguments specifying the output in the
		//     singular and plural cases.  by default "" and "s"
		"pluralize": func(items interface{}, pluralOverrides ...string) string {
			singular, plural := "", "s"
			if len(pluralOverrides) >= 1 {
				singular = pluralOverrides[0]
				if len(pluralOverrides) == 2 {
					plural = pluralOverrides[1]
				}
			}

			switch v := reflect.ValueOf(items); v.Kind() {
			case reflect.Int:
				if items.(int) != 1 {
					return plural
				}
			case reflect.Slice:
				if v.Len() != 1 {
					return plural
				}
			default:
				ERROR.Println("pluralize: unexpected type: ", v)
			}
			return singular
		},

		// Format a date according to the application's default date(time) format.
		"date": func(date time.Time) string {
			return date.Format(DateFormat)
		},
		"datetime": func(date time.Time) string {
			return date.Format(DateTimeFormat)
		},
		"slug": Slug,
	}
)

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
					// conform to expectations, so we wrap it in a func and handle those
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

            templatesAndEngine = new(templateAndEnvironment)
            if err = templatesAndEngine.setupTemplateEngine(); err == nil {
              err = templatesAndEngine.methods["initialAddAndParse"].(
                func(templateSet **abstractTemplateSet, templateName string, templateSource *string, basePath string) error)(
                  &templatesAndEngine.templateSet, templateName, &fileStr, basePath)
            }
          }()

          if funcError != nil {
						return funcError
					}

        } else {
          err = templatesAndEngine.methods["addAndParse"].(
            func(templateSet *abstractTemplateSet, templateName string, templateSource *string, basePath string) error)(
              templatesAndEngine.templateSet, templateName, &fileStr, basePath)
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

func (templatesAndEngine *templateAndEnvironment) setTemplateEngineName(templateEngineName string) (err error) {
  if templateEngineName == "" {
    templateEngineName = "GoTemplate"
  }
  for _, possibleName := range [...]string{"HAML", "GoTemplate"} {
    if possibleName == templateEngineName {
      templatesAndEngine.engineName = templateEngineName
    }
  }
  if templatesAndEngine.engineName == "" {
    err = fmt.Errorf("Unknown template engine name")
  }
  return
}

func (templatesAndEngine *templateAndEnvironment) setupTemplateEngine() (err error) {
  templateEngineName, _ := Config.String("template.engine")
  err = templatesAndEngine.setTemplateEngineName(templateEngineName)
  switch templatesAndEngine.engineName {
  case "HAML":
    templatesAndEngine.methods = map[string]interface{}{
      "initialAddAndParse": func(templateSet **abstractTemplateSet, templateName string, templateSource *string, basePath string) error {
        //HAMLTemplateSet := HAMLTemplate(*templateSet)
        return nil
      },
      "addAndParse": func(templateSet *abstractTemplateSet, templateName string, templateSource *string, basePath string) error {
        //HAMLTemplateSet := HAMLTemplate(*templateSet)
        return nil
      },
      "lookup": func(templateSet *abstractTemplateSet, templateName string, loader *TemplateLoader) *Template {
        //return HAMLTemplate{tmpl, loader}, err
        //HAMLTemplateSet := HAMLTemplate(*templateSet)
        return nil
      },
    }
  case "GoTemplate":
    templatesAndEngine.methods = map[string]interface{}{
      "initialAddAndParse": func(templateSet **abstractTemplateSet, templateName string, templateSource *string, basePath string) error {
        // Set the template delimiters for the project if present, then split into left
        // and right delimiters around a space character
        var splitDelims []string
        if TemplateDelims != "" {
          splitDelims = strings.Split(TemplateDelims, " ")
          if len(splitDelims) != 2 {
            log.Fatalln("app.conf: Incorrect format for template.delimiters")
          }
        }

        goTemplateSet := template.New(templateName).Funcs(TemplateFuncs)
        // If alternate delimiters set for the project, change them for this set
        if splitDelims != nil && basePath == ViewsPath {
          goTemplateSet.Delims(splitDelims[0], splitDelims[1])
        } else {
          goTemplateSet.Delims("", "")
        }
        goTemplateSet.Parse(*templateSource)
        var abstractTemplateSet abstractTemplateSet = *goTemplateSet
        *templateSet = &abstractTemplateSet
        return nil
      },
      "addAndParse": func(templateSet *abstractTemplateSet, templateName string, templateSource *string, basePath string) error {
        /*
        // If alternate delimiters set for the project, change them for this set
        if splitDelims != nil && basePath == ViewsPath {
          templateSet.Delims(splitDelims[0], splitDelims[1])
        } else {
          // Reset to default otherwise
          templateSet.Delims("", "")
        }
        */

        goTemplateSet := (*templateSet).(template.Template)
        goTemplateSet.New(templateName).Parse(*templateSource)
        var abstractTemplateSet abstractTemplateSet = goTemplateSet
        templateSet = &abstractTemplateSet
        return nil
      },
      "lookup": func(templateSet *abstractTemplateSet, templateName string, loader *TemplateLoader) *Template {
        goTemplateSet := (*templateSet).(template.Template)
        gotmpl := GoTemplate{&goTemplateSet, loader}
        var tmpl Template = gotmpl
        return &tmpl
      },
    }
  }
  return
}

// Return the Template with the given name.  The name is the template's path
// relative to a template loader root.
//
// An Error is returned if there was any problem with any of the templates.  (In
// this case, if a template is returned, it may still be usable.)
func (loader *TemplateLoader) Template(name string) (Template, error) {
	// Lower case the file name to support case-insensitive matching
	name = strings.ToLower(name)
  if loader.templatesAndEngine == nil {
    loader.templatesAndEngine = new(templateAndEnvironment)
    loader.templatesAndEngine.setupTemplateEngine()
  }
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

// Adapter for HAML Templates.
type HAMLTemplate struct {
	//*gohaml.Template
  template interface{}
	loader *TemplateLoader
}

func (haml HAMLTemplate) Name() string {
  return "my haml name"
}

// return a 'revel.Template' from Go's template.
func (haml HAMLTemplate) Render(wr io.Writer, arg interface{}) error {
  /*var scope = make(map[string]interface{})
  scope["lang"] = "HAML"
  engine, _ := gohaml.NewEngine(content)
  content := "I love <\n=lang<\n!"
  output := engine.Render(scope)
  */
  return nil
}

func (haml HAMLTemplate) Content() []string {
	content, _ := ReadLines(haml.loader.templatePaths[haml.Name()])
	return content
}

// Adapter for Go Templates.
type GoTemplate struct {
	*template.Template
	loader *TemplateLoader
}

// return a 'revel.Template' from Go's template.
func (gotmpl GoTemplate) Render(wr io.Writer, arg interface{}) error {
	return gotmpl.Execute(wr, arg)
}

func (gotmpl GoTemplate) Content() []string {
	content, _ := ReadLines(gotmpl.loader.templatePaths[gotmpl.Name()])
	return content
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
