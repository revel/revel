package revel

import (
	"fmt"
	"io"
	"html"
  "html/template"
	"log"
	"reflect"
	"strings"
	"time"
)

var (
  TemplateAPIOfGoTemplate = map[string]interface{}{
    "initialAddAndParse": func(templateSet **abstractTemplateSet, templateName string, templateSource *string, basePath string) (splitDelims []string, err error) {
      // Set the template delimiters for the project if present, then split into left
      // and right delimiters around a space character
      if TemplateDelims != "" {
        splitDelims = strings.Split(TemplateDelims, " ")
        if len(splitDelims) != 2 {
          log.Fatalln("app.conf: Incorrect format for template.delimiters")
        }
      }

      goTemplateSet := template.New(templateName).Funcs(TemplateFuncs)
      // If alternate delimiters set for the project, change them for this set
      if splitDelims != nil && basePath == ViewsPath {
        goTemplateSet.Delims(splitDelims[0], splitDelims[1])
      } else {
        goTemplateSet.Delims("", "")
      }
      goTemplateSet.Parse(*templateSource)
      var abstractTemplateSet abstractTemplateSet = *goTemplateSet
      *templateSet = &abstractTemplateSet
      return
    },
    "addAndParse": func(templateSet *abstractTemplateSet, templateName string, templateSource *string, basePath string, splitDelims []string) error {
      goTemplateSet := (*templateSet).(template.Template)

      // If alternate delimiters set for the project, change them for this set
      if splitDelims != nil && basePath == ViewsPath {
        goTemplateSet.Delims(splitDelims[0], splitDelims[1])
      } else {
        // Reset to default otherwise
        goTemplateSet.Delims("", "")
      }

      goTemplateSet.New(templateName).Parse(*templateSource)
      var abstractTemplateSet abstractTemplateSet = goTemplateSet
      templateSet = &abstractTemplateSet
      return nil
    },
    "lookup": func(templateSet *abstractTemplateSet, templateName string, loader *TemplateLoader) *Template {
      goTemplateSet := (*templateSet).(template.Template)
      gotmpl := GoTemplate{&goTemplateSet, loader}
      var tmpl Template = gotmpl
      return &tmpl
    },
  }

  // The functions available for use in the templates.
  TemplateFuncs = map[string]interface{}{
    "url": ReverseUrl,
    "eq":  Equal,
    "set": func(renderArgs map[string]interface{}, key string, value interface{}) template.HTML {
      renderArgs[key] = value
      return template.HTML("")
    },
    "append": func(renderArgs map[string]interface{}, key string, value interface{}) template.HTML {
      if renderArgs[key] == nil {
        renderArgs[key] = []interface{}{value}
      } else {
        renderArgs[key] = append(renderArgs[key].([]interface{}), value)
      }
      return template.HTML("")
    },
    "field": NewField,
    "option": func(f *Field, val, label string) template.HTML {
      selected := ""
      if f.Flash() == val {
        selected = " selected"
      }
      return template.HTML(fmt.Sprintf(`<option value="%s"%s>%s</option>`,
      html.EscapeString(val), selected, html.EscapeString(label)))
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
      return template.HTML(ERROR_CLASS)
    },

    "msg": func(renderArgs map[string]interface{}, message string, args ...interface{}) template.HTML {
      return template.HTML(Message(renderArgs[CurrentLocaleRenderArg].(string), message, args...))
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
  }
)

// Adapter for Go Templates.
type GoTemplate struct {
	*template.Template
	loader *TemplateLoader
}

// return a 'revel.Template' from Go's template.
func (gotmpl GoTemplate) Render(wr io.Writer, arg interface{}) error {
	return gotmpl.Execute(wr, arg)
}

func (gotmpl GoTemplate) Content() []string {
	content, _ := ReadLines(gotmpl.loader.templatePaths[gotmpl.Name()])
	return content
}
