package play

import (
	"html/template"
	"fmt"
	"path/filepath"
	"path"
	"os"
	"io/ioutil"
	"bytes"
	"strings"
	"regexp"
	"strconv"
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

var (
	// The functions available for use in the templates.
	tmplFuncs = map[string]interface{} {
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
			line := 0
			description := err.Error()
			ii := regexp.MustCompile(`:\d+:`).FindStringIndex(description)
			if ii != nil {
				line, err = strconv.Atoi(description[ii[0]+1:ii[1]-1])
				if err != nil {
					fmt.Println("Failed to parse line number from error message:", err)
				}
				description = description[ii[1]+1:]
			}
			return &CompileError{
				Title: "Template Compilation Error",
				Path: templateName,
				Description: description,
				Line: line,
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

// Executes a template and returns the result.
// name is the template path relative to views e.g. "Application/Index.html"
func (loader *TemplateLoader) RenderTemplate(name string, arg interface{}) (string, bool) {
	tmpl := loader.templateSet.Lookup(name)
	if tmpl == nil {
		return fmt.Sprintf("Template %s not found.\n", name), true
	}

	var b bytes.Buffer
	err := tmpl.Execute(&b, arg)
	if err != nil {
		return fmt.Sprintf("Failed to execute template: %s", err.Error()), true
	}

	return b.String(), false
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

	return router.Reverse(args[0].(string), args[1:]).Url
	return "#"
}

