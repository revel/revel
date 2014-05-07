package revel

import (
	"fmt"
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func init() {
	RegisterTemplateEnginer("default", NewGoTemplateEngine())
}

var (
	ERROR_CLASS = "hasError"

	invalidSlugPattern = regexp.MustCompile(`[^a-z0-9 _-]`)
	whiteSpacePattern  = regexp.MustCompile(`\s+`)

	// The default functions available for use in the templates.
	TemplateHelpers = template.FuncMap{
		// Return a url capable of invoking a given controller method:
		// "Application.ShowApp 123" => "/app/123"
		"url": func(args ...interface{}) (string, error) {
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
		},
		"eq": Equal,
		"set": func(renderArgs map[string]interface{}, key string, value interface{}) template.HTML {
			renderArgs[key] = value
			return template.JS("")
		},
		"append": func(renderArgs map[string]interface{}, key string, value interface{}) template.JS {
			if renderArgs[key] == nil {
				renderArgs[key] = []interface{}{value}
			} else {
				renderArgs[key] = append(renderArgs[key].([]interface{}), value)
			}
			return template.JS("")
		},
		"field": NewField,
		"firstof": func(args ...interface{}) interface{} {
			for _, val := range args {
				switch val.(type) {
				case nil:
					continue
				case string:
					if val == "" {
						continue
					}
					return val
				default:
					return val
				}
			}
			return nil
		},
		"option": func(f *Field, val interface{}, label string) template.HTML {
			selected := ""
			if f.Flash() == val || (f.Flash() == "" && f.Value() == val) {
				selected = " selected"
			}

			return template.HTML(fmt.Sprintf(`<option value="%s"%s>%s</option>`,
				html.EscapeString(fmt.Sprintf("%v", val)), selected, html.EscapeString(label)))
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
			str, ok := renderArgs[CurrentLocaleRenderArg].(string)
			if !ok {
				return ""
			}
			return template.HTML(Message(str, message, args...))
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
		"even": func(a int) bool { return (a % 2) == 0 },
		"slug": func(text string) string {
			separator := "-"
			text = strings.ToLower(text)
			text = invalidSlugPattern.ReplaceAllString(text, "")
			text = whiteSpacePattern.ReplaceAllString(text, separator)
			text = strings.Trim(text, separator)
			return text
		},
	}
)

type TemplateEnginer interface {
	Parse(s string) (*template.Template, error)
	SetHelpers(funcs template.FuncMap)
	WatchDir(dir os.FileInfo) bool
	WatchFile(file string) bool
}

type Template interface {
	Name() string
	Content() []string
	Render(wr io.Writer, arg interface{}) error
}

var _TEMPLATE_ENGINERS = map[string]TemplateEnginer{}

func RegisterTemplateEnginer(name string, enginer TemplateEnginer) {
	// avoid enginer overwriting
	if _, ok := _TEMPLATE_ENGINERS[name]; ok {
		return
	}

	_TEMPLATE_ENGINERS[name] = enginer
}

// This object handles loading and parsing of templates.
// Everything below the application's views directory is treated as a template.
type TemplateLoader struct {
	// template parse engine
	engine TemplateEnginer

	// If any error was encountered parsing the templates, it is stored here.
	engineError *Error

	// Paths to search for templates, in priority order.
	paths []string

	// This is the set of all templates under views
	templateSet *template.Template
	// Map from template name to the path from whence it was loaded.
	templatePaths map[string]string
}

func NewTemplateLoader(name string, paths []string) *TemplateLoader {
	if name == "" {
		name = "default"
	}

	enginer, ok := _TEMPLATE_ENGINERS[name]
	if !ok {
		panic(fmt.Sprintf("None template enginer found with name %s", name))
	}
	enginer.SetHelpers(TemplateHelpers)

	TRACE.Printf("New template loader with engine %s", name)

	return &TemplateLoader{
		engine:      enginer,
		paths:       paths,
		templateSet: template.New("").Funcs(TemplateHelpers),
	}
}

// This scans the views directory and parses all templates using engine as Go Templates.
// If a template fails to parse, the error is set on the loader.
// (It's awkward to refresh a single Go Template)
func (loader *TemplateLoader) Refresh() *Error {
	TRACE.Printf("Refreshing templates from %s", loader.paths)

	loader.engineError = nil
	loader.templatePaths = map[string]string{}

	// // Set the template delimiters for the project if present, then split into left
	// // and right delimiters around a space character
	// var splitDelims []string
	// if TemplateDelims != "" {
	// 	splitDelims = strings.Split(TemplateDelims, " ")
	// 	if len(splitDelims) != 2 {
	// 		log.Fatalln("app.conf: Incorrect format for template.delimiters")
	// 	}
	// }

	// Walk through the template loader's paths and build up a template set.
	for _, basePath := range loader.paths {
		// Walk only returns an error if the template loader is completely unusable
		// (namely, if one of the TemplateHelpers does not have an acceptable signature).
		walkErr := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				ERROR.Println("[filepath.Walk] ", path, " : ", err)
				return nil
			}

			// Walk into watchable directories
			if info.IsDir() {
				if !loader.engine.WatchDir(info) {
					return filepath.SkipDir
				}

				return nil
			}

			// Only add watchable
			if !loader.engine.WatchFile(info.Name()) {
				return nil
			}

			var templateString string

			// addTemplate allows the same template to be added multiple
			// times with different template names.
			addTemplate := func(name string) (err error) {
				// Convert template names to use forward slashes, even on Windows.
				if os.PathSeparator == '\\' {
					name = strings.Replace(name, `\`, `/`, -1)
				}

				// If we already loaded a template of this name, skip it.
				if _, ok := loader.templatePaths[name]; ok {
					return nil
				}
				loader.templatePaths[name] = path

				// Load the file if we haven't already
				if templateString == "" {
					fileBytes, err := ioutil.ReadFile(path)
					if err != nil {
						ERROR.Println("[ioutil.ReadFile] ", path, " : ", err.Error())
						return nil
					}

					templateString = string(fileBytes)
				}

				// Create the template set.
				// This panics if any of the funcs do not conform to expectations,
				// so we wrap it in a func and handle those panics by serving an error page.
				var (
					templateSet *template.Template
					engineErr   *Error
				)

				func() {
					defer func() {
						if err := recover(); err != nil {
							engineErr = &Error{
								Title:       "Panic (Template Loader)",
								Description: fmt.Sprintln(err),
							}
						}
					}()

					// TODO: shall we need config engine?
					// // If alternate delimiters set for the project, change them for this set
					// if splitDelims != nil && basePath == ViewsPath {
					// 	templateSet.Delims(splitDelims[0], splitDelims[1])
					// } else {
					// 	// Reset to default otherwise
					// 	templateSet.Delims("", "")
					// }

					templateSet, err = loader.engine.Parse(templateString)
					if err == nil {
						loader.AddTemplate(name, templateSet)
					}
				}()

				if engineErr != nil {
					return engineErr
				}

				return err
			}

			templateName := path[len(basePath)+1:]
			// Lower case the file name for case-insensitive matching
			lowerCaseTemplateName := strings.ToLower(templateName)

			err = addTemplate(templateName)
			err = addTemplate(lowerCaseTemplateName)

			// Store / report the first error encountered.
			if err != nil && loader.engineError == nil {
				_, line, description := parseTemplateError(err)

				loader.engineError = &Error{
					Title:       "Template Compilation Error",
					Line:        line,
					Path:        templateName,
					Description: description,
					SourceLines: strings.Split(templateString, "\n"),
				}

				ERROR.Printf("Template compilation error (In %s around line %d):\n%s", templateName, line, description)
			}

			return nil
		})

		// If there was an error with the Funcs, set it and return immediately.
		if walkErr != nil {
			loader.engineError = walkErr.(*Error)

			return loader.engineError
		}
	}

	// Note: engineError may or may not be set.
	return loader.engineError
}

// Create a new template with the given name and associate it with the loader.
// The name is the template's path relative to a template loader root.
//
// An error is returned if TemplateLoader has already been executed.
func (loader *TemplateLoader) AddTemplate(name string, tmpl *template.Template) error {
	_, err := loader.templateSet.AddParseTree(name, tmpl.Tree)
	return err
}

// Return the Template with the given name.
// The name is the template's path relative to a template loader root.
//
// An Error is returned if there was any problem with any of the templates.
// (In this case, if a template is returned, it may still be usable.)
func (loader *TemplateLoader) Template(name string) (Template, error) {
	// Lower case the file name to support case-insensitive matching
	name = strings.ToLower(name)

	// Look up and return the template.
	templateSet := loader.templateSet.Lookup(name)

	// This is necessary.
	// If a nil loader.engineError is returned directly, a caller testing against
	// nil will get the wrong result.  Something to do with casting *Error to error.
	var err error
	if loader.engineError != nil {
		err = loader.engineError
	}

	if templateSet == nil && err == nil {
		return nil, fmt.Errorf("Template %s not found.", name)
	}

	return GoTemplate{templateSet, loader}, err
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

//////////////////////
// Go template engine
//////////////////////
type GoTemplateEngine struct {
	driver  *template.Template
	helpers template.FuncMap
	counter uint32
}

func NewGoTemplateEngine() TemplateEnginer {
	return &GoTemplateEngine{
		driver:  template.New(""),
		helpers: template.FuncMap{},
		counter: 0,
	}
}

func (engine *GoTemplateEngine) Parse(s string) (*template.Template, error) {
	engine.counter += 1

	return engine.driver.New(fmt.Sprintf("[#%d] GO TEMPLATE ENGINE", engine.counter)).Parse(s)
}

func (engine *GoTemplateEngine) SetHelpers(helpers template.FuncMap) {
	for key, val := range helpers {
		engine.helpers[key] = val
	}

	engine.driver.Funcs(helpers)
}

func (engine *GoTemplateEngine) WatchDir(dir os.FileInfo) bool {
	// Watch all directories, except the ones starting with a dot.
	return !strings.HasPrefix(dir.Name(), ".")
}

func (engine GoTemplateEngine) WatchFile(file string) bool {
	// Watch all files with .html extension, except the ones starting with a dot.
	return !strings.HasPrefix(file, ".") && strings.HasSuffix(strings.ToLower(file), ".html")
}

/////////////////////
// Template helpers
/////////////////////

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
