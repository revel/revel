// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"fmt"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
)

// Error description, used as an argument to the error template.
type Error struct {
	SourceType               string   // The type of source that failed to build.
	Title, Path, Description string   // Description of the error, as presented to the user.
	Line, Column             int      // Where the error was encountered.
	SourceLines              []string // The entire source file, split into lines.
	Stack                    string   // The raw stack trace string from debug.Stack().
	MetaError                string   // Error that occurred producing the error page.
	Link                     string   // A configurable link to wrap the error source in
}

// SourceLine structure to hold the per-source-line details.
type SourceLine struct {
	Source  string
	Line    int
	IsError bool
}

// NewErrorFromPanic method finds the deepest stack from in user code and
// provide a code listing of that, on the line that eventually triggered
// the panic.  Returns nil if no relevant stack frame can be found.
func NewErrorFromPanic(err interface{}) *Error {

	// Parse the filename and line from the originating line of app code.
	// /Users/robfig/code/gocode/src/revel/examples/booking/app/controllers/hotels.go:191 (0x44735)
	stack := string(debug.Stack())
	frame, basePath := findRelevantStackFrame(stack)
	if frame == -1 {
		return nil
	}

	stack = stack[frame:]
	stackElement := stack[:strings.Index(stack, "\n")]
	colonIndex := strings.LastIndex(stackElement, ":")
	filename := stackElement[:colonIndex]
	var line int
	fmt.Sscan(stackElement[colonIndex+1:], &line)

	// Show an error page.
	description := "Unspecified error"
	if err != nil {
		description = fmt.Sprint(err)
	}
	lines, readErr := ReadLines(filename)
	if readErr != nil {
		utilLog.Error("Unable to read file", "file", filename, "error", readErr)
	}
	return &Error{
		Title:       "Runtime Panic",
		Path:        filename[len(basePath):],
		Line:        line,
		Description: description,
		SourceLines: lines,
		Stack:       stack,
	}
}

// Error method constructs a plaintext version of the error, taking
// account that fields are optionally set. Returns e.g. Compilation Error
// (in views/header.html:51): expected right delim in end; got "}"
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
	return fmt.Sprintf("%s%s Stack: %s", header, e.Description, e.Stack)
}

// ContextSource method returns a snippet of the source around
// where the error occurred.
func (e *Error) ContextSource() []SourceLine {
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

	lines := make([]SourceLine, end-start)
	for i, src := range e.SourceLines[start:end] {
		fileLine := start + i + 1
		lines[i] = SourceLine{src, fileLine, fileLine == e.Line}
	}
	return lines
}

// SetLink method prepares a link and assign to Error.Link attribute
func (e *Error) SetLink(errorLink string) {
	errorLink = strings.Replace(errorLink, "{{Path}}", e.Path, -1)
	errorLink = strings.Replace(errorLink, "{{Line}}", strconv.Itoa(e.Line), -1)

	e.Link = "<a href=" + errorLink + ">" + e.Path + ":" + strconv.Itoa(e.Line) + "</a>"
}

// Return the character index of the first relevant stack frame, or -1 if none were found.
// Additionally it returns the base path of the tree in which the identified code resides.
func findRelevantStackFrame(stack string) (int, string) {
	// Find first item in SourcePath that isn't in RevelPath.
	// If first item is in RevelPath, keep track of position, trim and check again.
	partialStack := stack
	sourcePath := filepath.ToSlash(SourcePath)
	revelPath := filepath.ToSlash(RevelPath)
	sumFrame := 0
	for {
		frame := strings.Index(partialStack, sourcePath)
		revelFrame := strings.Index(partialStack, revelPath)

		if frame == -1 {
			break
		} else if frame != revelFrame {
			return sumFrame + frame, SourcePath
		} else {
			// Need to at least trim off the first character so this frame isn't caught again.
			partialStack = partialStack[frame+1:]
			sumFrame += frame + 1
		}
	}
	for _, module := range Modules {
		if frame := strings.Index(stack, filepath.ToSlash(module.Path)); frame != -1 {
			return frame, module.Path
		}
	}
	return -1, ""
}
