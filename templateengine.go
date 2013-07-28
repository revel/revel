package revel

import (
	html "html/template"
	"io"
	text "text/template"
)

type Template interface {
	Name() string
	Execute(wr io.Writer, arg interface{}) error
}

// TemplateEngine parses templates and provides access to them.
type TemplateEngine interface {
	Lookup(name string) Template
	Parse(name, content string) error
	Delims(left, right string)
}

type HtmlTemplateEngine struct {
	*html.Template
}

func NewHtmlTemplateEngine() TemplateEngine {
	return &HtmlTemplateEngine{html.New("").Funcs(TemplateFuncs)}
}

func (e *HtmlTemplateEngine) Parse(name, content string) (err error) {
	_, err = e.Template.New(name).Parse(content)
	return
}

func (e *HtmlTemplateEngine) Lookup(name string) Template {
	if r := e.Template.Lookup(name); r != nil {
		return r
	}
	return nil
}

func (e *HtmlTemplateEngine) Delims(left, right string) {
	e.Template.Delims(left, right)
}

type TextTemplateEngine struct {
	*text.Template
}

func NewTextTemplateEngine() TemplateEngine {
	return &TextTemplateEngine{text.New("").Funcs(TemplateFuncs)}
}

func (e *TextTemplateEngine) Parse(name, content string) error {
	_, err := e.Template.New(name).Parse(content)
	return err
}

func (e *TextTemplateEngine) Lookup(name string) Template {
	if r := e.Template.Lookup(name); r != nil {
		return r
	}
	return nil
}

func (e *TextTemplateEngine) Delims(left, right string) {
	e.Template.Delims(left, right)
}
