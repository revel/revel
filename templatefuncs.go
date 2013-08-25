package revel

import (
	"fmt"
	"html"
	"html/template"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
)

var (
	ERROR_CLASS        = "hasError"
	invalidSlugPattern = regexp.MustCompile(`[^a-z0-9 _-]`)
	whiteSpacePattern  = regexp.MustCompile(`\s+`)

	// Funcs are available for use in templates.
	TemplateFuncs = map[string]interface{}{
		"url":        ReverseUrl,
		"eq":         Equal,
		"set":        setRenderArg,
		"append":     appendRenderArg,
		"field":      NewField,
		"option":     optionTemplate,
		"radio":      radioTemplate,
		"checkbox":   checkboxTemplate,
		"pad":        pad,
		"errorClass": errorClass,
		"msg":        msg,
		"nl2br":      nl2br,
		"raw":        raw,
		"pluralize":  Pluralize,
		"date":       formatDate,
		"datetime":   formatDatetime,
		"slug":       Slug,
	}
)

// Return a url capable of invoking a given controller method:
// "Application.ShowApp 123" => "/app/123"
func ReverseUrl(args ...interface{}) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no arguments provided to reverse route")
	}

	action := args[0].(string)
	actionSplit := strings.Split(action, ".")
	if len(actionSplit) != 2 {
		return "", fmt.Errorf("reversing '%s', expected 'Controller.Action'", action)
	}

	// Look up the types.
	var c Controller
	if err := c.SetAction(actionSplit[0], actionSplit[1]); err != nil {
		return "", fmt.Errorf("reversing %s: %s", action, err)
	}

	// Unbind the arguments.
	argsByName := make(map[string]string)
	for i, argValue := range args[1:] {
		Unbind(argsByName, c.MethodType.Args[i].Name, argValue)
	}

	return MainRouter.Reverse(args[0].(string), argsByName).Url, nil
}

func Slug(text string) string {
	separator := "-"
	text = strings.ToLower(text)
	text = invalidSlugPattern.ReplaceAllString(text, "")
	text = whiteSpacePattern.ReplaceAllString(text, separator)
	text = strings.Trim(text, separator)
	return text
}

// Pluralize, a helper for pluralizing words to correspond to data of dynamic length.
// items - a slice of items, or an integer indicating how many items there are.
// pluralOverrides - optional arguments specifying the output in the
//     singular and plural cases.  by default "" and "s"
func Pluralize(items interface{}, pluralOverrides ...string) string {
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
		glog.Errorln("pluralize: unexpected type: ", v)
	}
	return singular
}

func errorClass(name string, renderArgs map[string]interface{}) template.HTML {
	errorMap, ok := renderArgs["errors"].(map[string]*ValidationError)
	if !ok || errorMap == nil {
		glog.Warningln("Called 'errorClass' without 'errors' in the render args.")
		return template.HTML("")
	}
	valError, ok := errorMap[name]
	if !ok || valError == nil {
		return template.HTML("")
	}
	return template.HTML(ERROR_CLASS)
}

func setRenderArg(renderArgs map[string]interface{}, key string, value interface{}) template.HTML {
	renderArgs[key] = value
	return template.HTML("")
}

func appendRenderArg(renderArgs map[string]interface{}, key string, value interface{}) template.HTML {
	if renderArgs[key] == nil {
		renderArgs[key] = []interface{}{value}
	} else {
		renderArgs[key] = append(renderArgs[key].([]interface{}), value)
	}
	return template.HTML("")
}

func optionTemplate(f *Field, val, label string) template.HTML {
	selected := ""
	if f.Flash() == val {
		selected = " selected"
	}
	return template.HTML(fmt.Sprintf(`<option value="%s"%s>%s</option>`,
		html.EscapeString(val), selected, html.EscapeString(label)))
}

func radioTemplate(f *Field, val string) template.HTML {
	checked := ""
	if f.Flash() == val {
		checked = " checked"
	}
	return template.HTML(fmt.Sprintf(`<input type="radio" name="%s" value="%s"%s>`,
		html.EscapeString(f.Name), html.EscapeString(val), checked))
}

func checkboxTemplate(f *Field, val string) template.HTML {
	checked := ""
	if f.Flash() == val {
		checked = " checked"
	}
	return template.HTML(fmt.Sprintf(`<input type="checkbox" name="%s" value="%s"%s>`,
		html.EscapeString(f.Name), html.EscapeString(val), checked))
}

// Pad the given string with &nbsp;'s up to the given width.
func pad(str string, width int) template.HTML {
	if len(str) >= width {
		return template.HTML(html.EscapeString(str))
	}
	return template.HTML(html.EscapeString(str) + strings.Repeat("&nbsp;", width-len(str)))
}

func msg(renderArgs map[string]interface{}, message string, args ...interface{}) template.HTML {
	return template.HTML(Message(renderArgs[CurrentLocaleRenderArg].(string), message, args...))
}

// Replaces newlines with <br>
func nl2br(text string) template.HTML {
	return template.HTML(strings.Replace(template.HTMLEscapeString(text), "\n", "<br>", -1))
}

// Skips sanitation on the parameter.  Do not use with dynamic data.
func raw(text string) template.HTML {
	return template.HTML(text)
}

// Format a date according to the application's default date(time) format.
func formatDate(date time.Time) string {
	return date.Format(DateFormat)
}

func formatDatetime(date time.Time) string {
	return date.Format(DateTimeFormat)
}
