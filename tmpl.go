package rev

import (
	"errors"
	"fmt"
	"github.com/howeyc/fsnotify"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// This object handles loading and parsing of templates.
// Everything below the application's views directory is treated as a template.
type TemplateLoader struct {
	// This watcher watches the views tree for changes.
	watcher *fsnotify.Watcher
	// This is the set of all templates under views
	templateSet *template.Template
	// If an error was encountered parsing the templates, it is stored here.
	compileError *Error
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
			return template.HTML(
				fmt.Sprintf(`<option value="%s"%s>%s</option>`, val, selected, label))
		},
		"radio": func(f *Field, val string) template.HTML {
			checked := ""
			if f.Value == val {
				checked = " checked"
			}
			return template.HTML(
				fmt.Sprintf(`<input type="radio" name="%s" value="%s"%s>`, f.Name, val, checked))
		},
	}
)

func NewTemplateLoader() *TemplateLoader {
	// Watch all directories under /views
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		LOG.Fatal(err)
	}

	// Replace the unbuffered Event channel with a buffered one.
	// Otherwise multiple change events only come out one at a time, across
	// multiple page views.
	watcher.Event = make(chan *fsnotify.FileEvent, 10)
	watcher.Error = make(chan error, 10)

	// Walk through all files / directories under /views.
	// - Add each directory to the watcher.
	filepath.Walk(ViewsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			LOG.Println("Error walking views:", err)
			return nil
		}
		if info.IsDir() {
			err = watcher.Watch(path)
			if err != nil {
				LOG.Println("Failed to watch", path, ":", err)
			}
		}
		return nil
	})

	loader := &TemplateLoader{
		watcher: watcher,
	}
	loader.refresh()
	return loader
}

// This scans the views directory and parses all templates.
// If a template fails to parse, the error is set on the loader.
// (It's awkward to refresh a single Go Template )
func (loader *TemplateLoader) refresh() {
	LOG.Println("Refresh")
	loader.compileError = nil
	var templateSet *template.Template = nil
	walkErr := filepath.Walk(ViewsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			LOG.Println("error walking views:", err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		fileBytes, err := ioutil.ReadFile(path)
		if err != nil {
			LOG.Println("Failed reading file:", path)
			return nil
		}

		templateName := path[len(ViewsPath)+1:]
		fileStr := string(fileBytes)
		if templateSet == nil {
			templateSet, err = template.New(templateName).
				Funcs(Funcs).
				Parse(fileStr)
		} else {
			_, err = templateSet.New(templateName).Parse(fileStr)
		}

		if err != nil {
			line, description := parseTemplateError(err)
			return &Error{
				Title:       "Template Compilation Error",
				Path:        templateName,
				Description: description,
				Line:        line,
				SourceLines: strings.Split(fileStr, "\n"),
			}
		}
		return nil
	})

	if walkErr != nil {
		// There was an error parsing a template.
		// Log it to the console and return a friendly HTML error page.
		err := walkErr.(*Error)
		LOG.Printf("Template compilation error (In %s around line %d):\n%s",
			err.Path, err.Line, err.Description)
		loader.compileError = err
		return
	}

	loader.templateSet = templateSet
}

// Parse the line, and description from an error message like:
// html/template:Application/Register.html:36: no such template "footer.html"
func parseTemplateError(err error) (line int, description string) {
	description = err.Error()
	i := regexp.MustCompile(`:\d+:`).FindStringIndex(description)
	if i != nil {
		line, err = strconv.Atoi(description[i[0]+1 : i[1]-1])
		if err != nil {
			LOG.Println("Failed to parse line number from error message:", err)
		}
		description = description[i[1]+1:]
	}
	return line, description
}

func (loader *TemplateLoader) getTemplateContent(name string) ([]string, error) {
	return ReadLines(path.Join(ViewsPath, name))
}

func (loader *TemplateLoader) Template(name string) (Template, error) {
	// First, check to see if the watcher saw any changes.
	// Pull all pending events / errors from the watcher.
	refresh := false
	for {
		select {
		case ev := <-loader.watcher.Event:
			// Ignore changes to dotfiles.
			if !strings.HasPrefix(path.Base(ev.Name), ".") {
				refresh = true
			}
			continue
		case <-loader.watcher.Error:
			continue
		default:
			// No events left to pull
		}
		break
	}

	// If we got a qualifying event, refresh the templates.
	if refresh {
		loader.refresh()
	}

	// If there was an error refreshing the templates, return it.
	if loader.compileError != nil {
		return nil, loader.compileError
	}

	// Look up and return the template.
	tmpl := loader.templateSet.Lookup(name)
	if tmpl == nil {
		return nil, errors.New(fmt.Sprintf("Template %s not found.\n", name))
	}
	return GoTemplate{tmpl, loader}, nil
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
	content, _ := gotmpl.loader.getTemplateContent(gotmpl.Name())
	return content
}

/////////////////////
// Template functions
/////////////////////

// Return a url capable of invoking a given controller method:
// "Application.ShowApp 123" => "/app/123"
func ReverseUrl(args ...interface{}) string {
	if len(args) == 0 {
		LOG.Println("Warning: no arguments provided to url function")
		return "#"
	}

	action := args[0].(string)
	actionSplit := strings.Split(action, ".")
	var ctrl, meth string
	if len(actionSplit) != 2 {
		LOG.Println("Warning: Must provide Controller.Method for reverse router.")
		return "#"
	}
	ctrl, meth = actionSplit[0], actionSplit[1]
	controllerType := LookupControllerType(ctrl)
	methodType := controllerType.Method(meth)
	argsByName := make(map[string]string)
	for i, argValue := range args[1:] {
		argsByName[methodType.Args[i].Name] = fmt.Sprintf("%s", argValue)
	}

	return router.Reverse(args[0].(string), argsByName).Url
}
