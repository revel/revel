package play

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"path"
)

// A compilation error, used as an argument to the CompileError.html template.
type CompileError struct {
	SourceType               string   // The type of source that failed to build.
	Title, Path, Description string   // Description of the error, as presented to the user.
	Line, Column             int      // Where the error was encountered.
	SourceLines              []string // The entire source file, split into lines.
	MetaError                string   // Error that occurred producing the error page.
}

// An object to hold the per-source-line details.
type sourceLine struct {
	Source  string
	Line    int
	IsError bool
}

func (e CompileError) Error() string {
	return fmt.Sprintf("%s:%d: %s", e.Path, e.Line, e.Description)
}

// Returns a snippet of the source around where the error occurred.
func (e *CompileError) ContextSource() []sourceLine {
	if e.SourceLines == nil {
		return nil
	}
	start := (e.Line - 1) - 5
	if start < 0 {
		start = 0
	}
	end := (e.Line - 1) + 5
	if end > len(e.SourceLines) {
		end = len(e.SourceLines)
	}

	var lines []sourceLine = make([]sourceLine, end-start)
	for i, src := range e.SourceLines[start:end] {
		fileLine := start + i + 1
		lines[i] = sourceLine{src, fileLine, fileLine == e.Line}
	}
	return lines
}

var errorTemplate *template.Template

func (e *CompileError) Html() string {
	var b bytes.Buffer
	RenderError(&b, &e)
	return b.String()
}

func RenderError(buffer io.Writer, data interface{}) {
	if errorTemplate == nil {
		errorTemplate = template.Must(template.ParseFiles(
			path.Join(PlayTemplatePath, "CompileError.html")))
	}
	errorTemplate.Execute(buffer, &data)
}
