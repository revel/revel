package revel

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/xeonx/timeago"
	"html"
	"html/template"
	"reflect"
	"strings"
	"time"
)

var (
	// The functions available for use in the templates.
	TemplateFuncs = map[string]interface{}{
		"url": ReverseURL,
		"set": func(viewArgs map[string]interface{}, key string, value interface{}) template.JS {
			viewArgs[key] = value
			return template.JS("")
		},
		"append": func(viewArgs map[string]interface{}, key string, value interface{}) template.JS {
			if viewArgs[key] == nil {
				viewArgs[key] = []interface{}{value}
			} else {
				viewArgs[key] = append(viewArgs[key].([]interface{}), value)
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

		"errorClass": func(name string, viewArgs map[string]interface{}) template.HTML {
			errorMap, ok := viewArgs["errors"].(map[string]*ValidationError)
			if !ok || errorMap == nil {
				templateLog.Warn("errorClass: Called 'errorClass' without 'errors' in the view args.")
				return template.HTML("")
			}
			valError, ok := errorMap[name]
			if !ok || valError == nil {
				return template.HTML("")
			}
			return template.HTML(ErrorCSSClass)
		},

		"msg": func(viewArgs map[string]interface{}, message string, args ...interface{}) template.HTML {
			str, ok := viewArgs[CurrentLocaleViewArg].(string)
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
				templateLog.Error("pluralize: unexpected type: ", "value", v)
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
		// Fetch an object from the session.
		"session": func(key string, viewArgs map[string]interface{}) interface{} {
			if viewArgs != nil {
				if c, found := viewArgs["_controller"]; found {
					if v, err := c.(*Controller).Session.Get(key); err == nil {
						return v
					} else {
						templateLog.Errorf("template.session, key %s error %v", key, err)
					}
				} else {
					templateLog.Warnf("template.session, key %s requested without controller", key)
				}
			} else {
				templateLog.Warnf("template.session, key %s requested passing in view args", key)
			}
			return ""
		},

		"slug": Slug,
		"even": func(a int) bool { return (a % 2) == 0 },

		// Using https://github.com/xeonx/timeago
		"timeago": TimeAgo,
		"i18ntemplate": func(args ...interface{}) (template.HTML, error) {
			templateName, lang := "", ""
			var viewArgs interface{}
			switch len(args) {
			case 0:
				templateLog.Error("i18ntemplate: No arguments passed to template call")
			case 1:
				// Assume only the template name is passed in
				templateName = args[0].(string)
			case 2:
				// Assume template name and viewArgs is passed in
				templateName = args[0].(string)
				viewArgs = args[1]
				// Try to extract language from the view args
				if viewargsmap, ok := viewArgs.(map[string]interface{}); ok {
					lang, _ = viewargsmap[CurrentLocaleViewArg].(string)
				}
			default:
				// Assume third argument is the region
				templateName = args[0].(string)
				viewArgs = args[1]
				lang, _ = args[2].(string)
				if len(args) > 3 {
					templateLog.Error("i18ntemplate: Received more parameters then needed for", "template", templateName)
				}
			}

			var buf bytes.Buffer
			// Get template
			tmpl, err := MainTemplateLoader.TemplateLang(templateName, lang)
			if err == nil {
				err = tmpl.Render(&buf, viewArgs)
			} else {
				templateLog.Error("i18ntemplate: Failed to render i18ntemplate ", "name", templateName, "error", err)
			}
			return template.HTML(buf.String()), err
		},
	}
)

/////////////////////
// Template functions
/////////////////////

// ReverseURL returns a url capable of invoking a given controller method:
// "Application.ShowApp 123" => "/app/123"
func ReverseURL(args ...interface{}) (template.URL, error) {
	if len(args) == 0 {
		return "", errors.New("no arguments provided to reverse route")
	}

	action := args[0].(string)
	if action == "Root" {
		return template.URL(AppRoot), nil
	}

	pathData, found := splitActionPath(nil, action, true)

	if !found {
		return "", fmt.Errorf("reversing '%s', expected 'Controller.Action'", action)
	}

	// Look up the types.

	if pathData.TypeOfController == nil {
		return "", fmt.Errorf("Failed reversing %s: controller not found %#v", action, pathData)
	}

	// Note method name is case insensitive search
	methodType := pathData.TypeOfController.Method(pathData.MethodName)
	if methodType == nil {
		return "", errors.New("revel/controller: In " + action + " failed to find function " + pathData.MethodName)
	}

	if len(methodType.Args) < len(args)-1 {
		return "", fmt.Errorf("reversing %s: route defines %d args, but received %d",
			action, len(methodType.Args), len(args)-1)
	}
	// Unbind the arguments.
	argsByName := make(map[string]string)
	// Bind any static args first
	fixedParams := len(pathData.FixedParamsByName)

	for i, argValue := range args[1:] {
		Unbind(argsByName, methodType.Args[i+fixedParams].Name, argValue)
	}

	return template.URL(MainRouter.Reverse(args[0].(string), argsByName).URL), nil
}

func Slug(text string) string {
	separator := "-"
	text = strings.ToLower(text)
	text = invalidSlugPattern.ReplaceAllString(text, "")
	text = whiteSpacePattern.ReplaceAllString(text, separator)
	text = strings.Trim(text, separator)
	return text
}

var timeAgoLangs = map[string]timeago.Config{}

func TimeAgo(args ...interface{}) string {

	datetime := time.Now()
	lang := ""
	var viewArgs interface{}
	switch len(args) {
	case 0:
		templateLog.Error("TimeAgo: No arguments passed to timeago")
	case 1:
		// only the time is passed in
		datetime = args[0].(time.Time)
	case 2:
		// time and region is passed in
		datetime = args[0].(time.Time)
		switch v := reflect.ValueOf(args[1]); v.Kind() {
		case reflect.String:
			// second params type string equals region
			lang, _ = args[1].(string)
		case reflect.Map:
			// second params type map equals viewArgs
			viewArgs = args[1]
			if viewargsmap, ok := viewArgs.(map[string]interface{}); ok {
				lang, _ = viewargsmap[CurrentLocaleViewArg].(string)
			}
		default:
			templateLog.Error("TimeAgo: unexpected type: ", "value", v)
		}
	default:
		// Assume third argument is the region
		datetime = args[0].(time.Time)
		if reflect.ValueOf(args[1]).Kind() != reflect.Map {
			templateLog.Error("TimeAgo: unexpected type", "value", args[1])
		}
		if reflect.ValueOf(args[2]).Kind() != reflect.String {
			templateLog.Error("TimeAgo: unexpected type: ", "value", args[2])
		}
		viewArgs = args[1]
		lang, _ = args[2].(string)
		if len(args) > 3 {
			templateLog.Error("TimeAgo: Received more parameters then needed for timeago")
		}
	}
	if lang == "" {
		lang, _ = Config.String(defaultLanguageOption)
		if lang == "en" {
			timeAgoLangs[lang] = timeago.English
		}
	}
	_, ok := timeAgoLangs[lang]
	if !ok {
		timeAgoLangs[lang] = timeago.Config{
			PastPrefix:   "",
			PastSuffix:   " " + MessageFunc(lang, "ago"),
			FuturePrefix: MessageFunc(lang, "in") + " ",
			FutureSuffix: "",
			Periods: []timeago.FormatPeriod{
				{time.Second, MessageFunc(lang, "about a second"), MessageFunc(lang, "%d seconds")},
				{time.Minute, MessageFunc(lang, "about a minute"), MessageFunc(lang, "%d minutes")},
				{time.Hour, MessageFunc(lang, "about an hour"), MessageFunc(lang, "%d hours")},
				{timeago.Day, MessageFunc(lang, "one day"), MessageFunc(lang, "%d days")},
				{timeago.Month, MessageFunc(lang, "one month"), MessageFunc(lang, "%d months")},
				{timeago.Year, MessageFunc(lang, "one year"), MessageFunc(lang, "%d years")},
			},
			Zero:          MessageFunc(lang, "about a second"),
			Max:           73 * time.Hour,
			DefaultLayout: "2006-01-02",
		}

	}
	return timeAgoLangs[lang].Format(datetime)
}
