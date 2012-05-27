package rev

import "fmt"

// An error description, used as an argument to the error template.
type Error struct {
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

// Construct a plaintext version of the error, taking account that fields are optionally set.
// Returns e.g. Compilation Error (in views/header.html:51): expected right delim in end; got "}"
func (e *Error) Error() string {
	loc := ""
	if e.Path != "" {
		line := ""
		if e.Line != 0 {
			line = fmt.Sprintf(":%d", e.Line)
		}
		loc = fmt.Sprintf("(in %s%s)", e.Path, line)
	}
	header := loc
	if e.Title != "" {
		if loc != "" {
			header = fmt.Sprintf("%s %s: ", e.Title, loc)
		} else {
			header = fmt.Sprintf("%s: ", e.Title)
		}
	}
	return fmt.Sprintf("%s%s", header, e.Description)
}

// Returns a snippet of the source around where the error occurred.
func (e *Error) ContextSource() []sourceLine {
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
