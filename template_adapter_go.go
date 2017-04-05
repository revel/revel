package revel

import (
	"fmt"
	"html"
	"html/template"
	"io"
	"log"
	"reflect"
	"strings"
	"time"
)

const GO_TEMPLATE = "go"

var (
	// The functions available for use in the templates.
	TemplateFuncs = map[string]interface{}{
		"url": ReverseURL,
		"set": func(renderArgs map[string]interface{}, key string, value interface{}) template.JS {
			renderArgs[key] = value
			return template.JS("")
		},
		"append": func(renderArgs map[string]interface{}, key string, value interface{}) template.JS {
			if renderArgs[key] == nil {
				renderArgs[key] = []interface{}{value}
			} else {
				renderArgs[key] = append(renderArgs[key].([]interface{}), value)
			}
			return template.JS("")
		},
		"field": NewField,
		"firstof": func(args ...interface{}) interface{} {
			for _, val := range args {
				switch val.(type) {
				case nil:
					continue
				case string:
					if val == "" {
						continue
					}
					return val
				default:
					return val
				}
			}
			return nil
		},
		"option": func(f *Field, val interface{}, label string) template.HTML {
			selected := ""
			if f.Flash() == val || (f.Flash() == "" && f.Value() == val) {
				selected = " selected"
			}

			return template.HTML(fmt.Sprintf(`<option value="%s"%s>%s</option>`,
				html.EscapeString(fmt.Sprintf("%v", val)), selected, html.EscapeString(label)))
		},
		"radio": func(f *Field, val string) template.HTML {
			checked := ""
			if f.Flash() == val {
				checked = " checked"
			}
			return template.HTML(fmt.Sprintf(`<input type="radio" name="%s" value="%s"%s>`,
				html.EscapeString(f.Name), html.EscapeString(val), checked))
		},
		"checkbox": func(f *Field, val string) template.HTML {
			checked := ""
			if f.Flash() == val {
				checked = " checked"
			}
			return template.HTML(fmt.Sprintf(`<input type="checkbox" name="%s" value="%s"%s>`,
				html.EscapeString(f.Name), html.EscapeString(val), checked))
		},
		// Pads the given string with &nbsp;'s up to the given width.
		"pad": func(str string, width int) template.HTML {
			if len(str) >= width {
				return template.HTML(html.EscapeString(str))
			}
			return template.HTML(html.EscapeString(str) + strings.Repeat("&nbsp;", width-len(str)))
		},

		"errorClass": func(name string, renderArgs map[string]interface{}) template.HTML {
			errorMap, ok := renderArgs["errors"].(map[string]*ValidationError)
			if !ok || errorMap == nil {
				WARN.Println("Called 'errorClass' without 'errors' in the render args.")
				return template.HTML("")
			}
			valError, ok := errorMap[name]
			if !ok || valError == nil {
				return template.HTML("")
			}
			return template.HTML(ErrorCSSClass)
		},

		"msg": func(renderArgs map[string]interface{}, message string, args ...interface{}) template.HTML {
			str, ok := renderArgs[CurrentLocaleRenderArg].(string)
			if !ok {
				return ""
			}
			return template.HTML(MessageFunc(str, message, args...))
		},

		// Replaces newlines with <br>
		"nl2br": func(text string) template.HTML {
			return template.HTML(strings.Replace(template.HTMLEscapeString(text), "\n", "<br>", -1))
		},

		// Skips sanitation on the parameter.  Do not use with dynamic data.
		"raw": func(text string) template.HTML {
			return template.HTML(text)
		},

		// Pluralize, a helper for pluralizing words to correspond to data of dynamic length.
		// items - a slice of items, or an integer indicating how many items there are.
		// pluralOverrides - optional arguments specifying the output in the
		//     singular and plural cases.  by default "" and "s"
		"pluralize": func(items interface{}, pluralOverrides ...string) string {
			singular, plural := "", "s"
			if len(pluralOverrides) >= 1 {
				singular = pluralOverrides[0]
				if len(pluralOverrides) == 2 {
					plural = pluralOverrides[1]
				}
			}

			switch v := reflect.ValueOf(items); v.Kind() {
			case reflect.Int:
				if items.(int) != 1 {
					return plural
				}
			case reflect.Slice:
				if v.Len() != 1 {
					return plural
				}
			default:
				ERROR.Println("pluralize: unexpected type: ", v)
			}
			return singular
		},

		// Format a date according to the application's default date(time) format.
		"date": func(date time.Time) string {
			return date.Format(DateFormat)
		},
		"datetime": func(date time.Time) string {
			return date.Format(DateTimeFormat)
		},
		"slug": Slug,
		"even": func(a int) bool { return (a % 2) == 0 },
	}
)

// Adapter for Go Templates.
type GoTemplate struct {
	*template.Template
	engine *GoEngine
	*BaseTemplate
}

// return a 'revel.Template' from Go's template.
func (gotmpl GoTemplate) Render(wr io.Writer, arg interface{}) error {
	return gotmpl.Execute(wr, arg)
}

type GoEngine struct {
	loader      *TemplateLoader
	templateSet *template.Template
	// TemplatesBylowerName is a map from lower case template name to the real template.
	templatesBylowerName map[string]*GoTemplate
	splitDelims          []string
	BaseTemplateEngine
}

func (engine *GoEngine) ParseAndAdd(baseTemplate *BaseTemplate) error {
	// If alternate delimiters set for the project, change them for this set
	if engine.splitDelims != nil && baseTemplate.Location() == ViewsPath {
		engine.templateSet.Delims(engine.splitDelims[0], engine.splitDelims[1])
	} else {
		// Reset to default otherwise
		engine.templateSet.Delims("", "")
	}
	templateSource := string(baseTemplate.FileBytes)
	baseTemplate.TemplateName = engine.ConvertPath(baseTemplate.TemplateName)
	tpl, err := engine.templateSet.New(baseTemplate.TemplateName).Parse(templateSource)
	if nil != err {
		_, line, description := ParseTemplateError(err)
		return &Error{
			Title:       "Template Compilation Error",
			Path:        baseTemplate.TemplateName,
			Description: description,
			Line:        line,
			SourceLines: strings.Split(templateSource, "\n"),
		}
	}
	engine.templatesBylowerName[baseTemplate.TemplateName] = &GoTemplate{Template: tpl, engine: engine, BaseTemplate: baseTemplate}
	return nil
}

func (engine *GoEngine) Lookup(templateName string) Template {
	// Case-insensitive matching of template file name

	if tpl, found := engine.templatesBylowerName[engine.ConvertPath(templateName)]; found {
		return tpl
	}
	return nil
}
func (engine *GoEngine) Name() string {
	return GO_TEMPLATE
}
func (engine *GoEngine) Event(action int, i interface{}) {
	if action == TEMPLATE_REFRESH {
		// At this point all the templates have been passed into the
		engine.templatesBylowerName = map[string]*GoTemplate{}
		engine.templateSet = template.New("__root__").Funcs(TemplateFuncs)
		// Check to see what should be used for case sensitivity
		engine.CaseInsensitiveMode = Config.StringDefault("go.template.path", "lower") != "case"
	}
}
func init() {
	RegisterTemplateLoader(GO_TEMPLATE, func(loader *TemplateLoader) (TemplateEngine, error) {
		// Set the template delimiters for the project if present, then split into left
		// and right delimiters around a space character

		TemplateDelims := Config.StringDefault("template.go.delimiters", "")
		var splitDelims []string
		if TemplateDelims != "" {
			splitDelims = strings.Split(TemplateDelims, " ")
			if len(splitDelims) != 2 {
				log.Fatalln("app.conf: Incorrect format for template.delimiters")
			}
		}

		return &GoEngine{
			loader:               loader,
			templateSet:          template.New("__root__").Funcs(TemplateFuncs),
			templatesBylowerName: map[string]*GoTemplate{},
			splitDelims:          splitDelims,
		}, nil
	})
}
