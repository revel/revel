package play

import (
	"errors"
	"fmt"
	"github.com/robfig/fsnotify"
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

var (
	// The functions available for use in the templates.
	tmplFuncs = map[string]interface{}{
		"url": ReverseUrl,
		"field": func(name string, renderArgs map[string]interface{}) *Field {
			value, _ := renderArgs["flash"].(map[string]string)[name]
			err, _ := renderArgs["errors"].(map[string]*ValidationError)[name]
			return &Field{
				Name:  name,
				Value: value,
				Error: err,
			}
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
	loader.Refresh()
	return loader
}

// This scans the views directory and parses all templates.
// If a template fails to parse, the error is returned.
// (It's awkward to refresh a single Go Template )
func (loader *TemplateLoader) Refresh() (err *CompileError) {
	LOG.Println("Refresh")
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
				Funcs(tmplFuncs).
				Parse(fileStr)
		} else {
			_, err = templateSet.New(templateName).Parse(fileStr)
		}

		if err != nil {
			line, description := parseTemplateError(err)
			return &CompileError{
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
		err = walkErr.(*CompileError)
	}

	loader.templateSet = templateSet

	// There was an error parsing a template.
	// Log it to the console and return a friendly HTML error page.
	if err != nil {
		LOG.Printf("Template compilation error (In %s around line %d):\n%s",
			err.Path, err.Line, err.Description)
		return err
	}

	return nil
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
		loader.Refresh()
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

// return a 'play.Template' from Go's template.
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
		argsByName[methodType.Args[i].Name] = argValue.(string)
	}

	return router.Reverse(args[0].(string), argsByName).Url
	return "#"
}
