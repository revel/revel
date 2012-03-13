package play

import (
	"errors"
	"fmt"
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
	// This is the set of all templates under views
	templateSet *template.Template
	// If a template failed to parse, this holds the error.
	// (All templates must parse before the TemplateLoader can be used)
	error *template.Error

	viewsDir string
}

type Template interface {
	Name() string
	Content() []string
	Render(wr io.Writer, arg interface{}) error
}

var (
	// The functions available for use in the templates.
	tmplFuncs = map[string]interface{}{
		"url": ReverseUrl,
	}
)

// This scans the views directory and parses all templates.
// If a template fails to parse, the error is returned.
func (loader *TemplateLoader) LoadTemplates() (err *CompileError) {
	viewsDir := path.Join(AppPath, "views")
	var templateSet *template.Template = nil
	walkErr := filepath.Walk(viewsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			LOG.Printf("%v", err)
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// The template name is the filename relative to the views directory.
		templateName := path[len(viewsDir)+1:]
		fileBytes, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Println("Failed reading file:", path)
			return nil
		}

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
	loader.viewsDir = viewsDir

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
	return ReadLines(path.Join(loader.viewsDir, name))
}

func (loader *TemplateLoader) Template(name string) (Template, error) {
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
	content, _ :=  gotmpl.loader.getTemplateContent(gotmpl.Name())
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
	ctrl, meth := actionSplit[0], actionSplit[1]
	controllerType := LookupControllerType(ctrl)
	methodType := controllerType.Method(meth)
	argsByName := make(map[string]string)
	for i, argValue := range args[1:] {
		argsByName[methodType.Args[i].Name] = argValue.(string)
	}

	return router.Reverse(args[0].(string), argsByName).Url
	return "#"
}
