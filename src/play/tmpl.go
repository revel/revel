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

// type Template struct {
// 	template *template.Template
// }

type sourceLine struct {
	Source string
	Line int
	IsError bool
}

type TemplateCompilationError struct {
	Title string
	Path string
	Line int
	Description string
	SourceLines []string
}

func (e TemplateCompilationError) Error() string {
	return fmt.Sprintf("html/template:%s:%d: %s", e.Path, e.Line, e.Description)
}

// Returns a snippet of the source around where the error occurred.
func (e TemplateCompilationError) ContextSource() []sourceLine {
	start := (e.Line - 1) - 5
	if start < 0 {
		start = 0
	}
	end := (e.Line - 1) + 5
	if end > len(e.SourceLines) {
		end = len(e.SourceLines)
	}
	var lines []sourceLine = make([]sourceLine, end - start)
	for i, src := range(e.SourceLines[start:end]) {
		lines[i] = sourceLine{src, start+i+1, i == e.Line - 1}
	}
	return lines
}

var errorTemplate = template.Must(template.ParseFiles(filepath.Join(PlayPath, "error.html")))

func (e TemplateCompilationError) Html() string {
	var b bytes.Buffer
	errorTemplate.Execute(&b, &e)
	return b.String()
}

// This scans the views directory and parses all templates.
// If a template fails to parse, the error is returned.
func (loader *TemplateLoader) LoadTemplates() (err *TemplateCompilationError) {
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
			templateSet, err = template.New(templateName).Parse(fileStr)
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
			return &TemplateCompilationError{
				"Template Compilation Error",
				templateName,
				line,
				description,
				strings.Split(fileStr, "\n"),
			}
		}
		return nil
	})

	if walkErr != nil {
		err = walkErr.(*TemplateCompilationError)
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



