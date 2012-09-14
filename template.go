package rev

import (
	"fmt"
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// This object handles loading and parsing of templates.
// Everything below the application's views directory is treated as a template.
type TemplateLoader struct {
	// This is the set of all templates under views
	templateSet *template.Template
	// If an error was encountered parsing the templates, it is stored here.
	compileError *Error
	// Paths to search for templates, in priority order.
	paths []string
	// Map from template name to the path from whence it was loaded.
	templatePaths map[string]string
}

type Template interface {
	Name() string
	Content() []string
	Render(wr io.Writer, arg interface{}) error
}

type Field struct {
	Name, Value string
	Error       *ValidationError
}

func (f *Field) ErrorClass() string {
	if f.Error != nil {
		return "hasError"
	}
	return ""
}

// Return "checked" if this field.Value matches the provided value
func (f *Field) Checked(val string) string {
	if f.Value == val {
		return "checked"
	}
	return ""
}

var (
	// The functions available for use in the templates.
	Funcs = map[string]interface{}{
		"url": ReverseUrl,
		"eq":  func(a, b interface{}) bool { return a == b },
		"set": func(key string, value interface{}, renderArgs map[string]interface{}) template.HTML {
			renderArgs[key] = value
			return template.HTML("")
		},
		"append": func(key string, value interface{}, renderArgs map[string]interface{}) template.HTML {
			if renderArgs[key] == nil {
				renderArgs[key] = []interface{}{value}
			} else {
				renderArgs[key] = append(renderArgs[key].([]interface{}), value)
			}
			return template.HTML("")
		},
		"field": func(name string, renderArgs map[string]interface{}) *Field {
			value, _ := renderArgs["flash"].(map[string]string)[name]
			err, _ := renderArgs["errors"].(map[string]*ValidationError)[name]
			return &Field{
				Name:  name,
				Value: value,
				Error: err,
			}
		},
		"option": func(f *Field, val, label string) template.HTML {
			selected := ""
			if f.Value == val {
				selected = " selected"
			}
			return template.HTML(fmt.Sprintf(`<option value="%s"%s>%s</option>`,
				html.EscapeString(val), selected, html.EscapeString(label)))
		},
		"radio": func(f *Field, val string) template.HTML {
			checked := ""
			if f.Value == val {
				checked = " checked"
			}
			return template.HTML(fmt.Sprintf(`<input type="radio" name="%s" value="%s"%s>`,
				html.EscapeString(f.Name), html.EscapeString(val), checked))
		},

		// Pads the given string with &nbsp;'s up to the given width.
		"pad": func(str string, width int) template.HTML {
			if len(str) >= width {
				return template.HTML(html.EscapeString(str))
			}
			return template.HTML(html.EscapeString(str) + strings.Repeat("&nbsp;", width-len(str)))
		},
	}
)

func NewTemplateLoader(paths ...string) *TemplateLoader {
	loader := &TemplateLoader{
		paths: paths,
	}
	loader.Refresh()
	return loader
}

// This scans the views directory and parses all templates as Go Templates.
// If a template fails to parse, the error is set on the loader.
// (It's awkward to refresh a single Go Template)
func (loader *TemplateLoader) Refresh() *Error {
	TRACE.Println("Refresh")
	loader.compileError = nil
	loader.templatePaths = map[string]string{}

	// Walk through the template loader's paths and build up a template set.
	var templateSet *template.Template = nil
	for _, basePath := range loader.paths {

		// Walk only returns an error if the template loader is completely unusable
		// (namely, if one of the Funcs does not have an acceptable signature).
		funcErr := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				ERROR.Println("error walking templates:", err)
				return nil
			}

			// Walk into directories.
			if info.IsDir() {
				return nil
			}

			// If we already loaded a template of this name, skip it.
			templateName := path[len(basePath)+1:]
			if runtime.GOOS == "windows" {
				templateName = strings.Replace(templateName, `\`, `/`, -1)
			}
			if _, ok := loader.templatePaths[templateName]; ok {
				return nil
			}
			loader.templatePaths[templateName] = path

			fileBytes, err := ioutil.ReadFile(path)
			if err != nil {
				ERROR.Println("Failed reading file:", path)
				return nil
			}

			fileStr := string(fileBytes)
			if templateSet == nil {
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
					templateSet = template.New(templateName).Funcs(Funcs)
					_, err = templateSet.Parse(fileStr)
				}()

				if funcError != nil {
					return funcError
				}

			} else {
				_, err = templateSet.New(templateName).Parse(fileStr)
			}

			// Store / report the first error encountered.
			if err != nil && loader.compileError == nil {
				line, description := parseTemplateError(err)
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
	loader.templateSet = templateSet
	return loader.compileError
}

// Parse the line, and description from an error message like:
// html/template:Application/Register.html:36: no such template "footer.html"
func parseTemplateError(err error) (line int, description string) {
	description = err.Error()
	i := regexp.MustCompile(`:\d+:`).FindStringIndex(description)
	if i != nil {
		line, err = strconv.Atoi(description[i[0]+1 : i[1]-1])
		if err != nil {
			ERROR.Println("Failed to parse line number from error message:", err)
		}
		description = description[i[1]+1:]
	}
	return line, description
}

// Return the Template with the given name.  The name is the template's path
// relative to a template loader root.
//
// An Error is returned if there was any problem with any of the templates.  (In
// this case, if a template is returned, it may still be usable.)
func (loader *TemplateLoader) Template(name string) (Template, error) {
	// Look up and return the template.
	tmpl := loader.templateSet.Lookup(name)

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

	return GoTemplate{tmpl, loader}, err
}

// Adapter for Go Templates.
type GoTemplate struct {
	*template.Template
	loader *TemplateLoader
}

// return a 'rev.Template' from Go's template.
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
func ReverseUrl(args ...interface{}) string {
	if len(args) == 0 {
		ERROR.Println("Warning: no arguments provided to url function")
		return "#"
	}

	action := args[0].(string)
	actionSplit := strings.Split(action, ".")
	var ctrl, meth string
	if len(actionSplit) != 2 {
		ERROR.Println("Warning: Must provide Controller.Method for reverse router.")
		return "#"
	}
	ctrl, meth = actionSplit[0], actionSplit[1]
	controllerType := LookupControllerType(ctrl)
	methodType := controllerType.Method(meth)
	argsByName := make(map[string]string)
	for i, argValue := range args[1:] {
		argsByName[methodType.Args[i].Name] = fmt.Sprintf("%s", argValue)
	}

	return MainRouter.Reverse(args[0].(string), argsByName).Url
}
