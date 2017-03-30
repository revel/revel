// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/revel/config"
)

const (
	// CurrentLocaleRenderArg the key for the current locale render arg value
	CurrentLocaleRenderArg = "currentLocale"

	messageFilesDirectory  = "messages"
	messageFilePattern     = `^\w+\.[a-zA-Z]{2}$`
	defaultUnknownFormat   = "??? %s ???"
	unknownFormatConfigKey = "i18n.unknown_format"
	defaultLanguageOption  = "i18n.default_language"
	localeCookieConfigKey  = "i18n.cookie"
)

var (
	// All currently loaded message configs.
	messages map[string]*config.Config
)

// MessageFunc allows you to override the translation interface.
//
// Set this to your own function that translates to the current locale.
// This allows you to set up your own loading and logging of translated texts.
//
// See Message(...) in i18n.go for example of function.
var MessageFunc = Message

// MessageLanguages returns all currently loaded message languages.
func MessageLanguages() []string {
	languages := make([]string, len(messages))
	i := 0
	for language := range messages {
		languages[i] = language
		i++
	}
	return languages
}

// Message performs a message look-up for the given locale and message using the given arguments.
//
// When either an unknown locale or message is detected, a specially formatted string is returned.
func Message(locale, message string, args ...interface{}) string {
	language, region := parseLocale(locale)
	unknownValueFormat := getUnknownValueFormat()

	messageConfig, knownLanguage := messages[language]
	if !knownLanguage {
		TRACE.Printf("Unsupported language for locale '%s' and message '%s', trying default language", locale, message)

		if defaultLanguage, found := Config.String(defaultLanguageOption); found {
			TRACE.Printf("Using default language '%s'", defaultLanguage)

			messageConfig, knownLanguage = messages[defaultLanguage]
			if !knownLanguage {
				WARN.Printf("Unsupported default language for locale '%s' and message '%s'", defaultLanguage, message)
				return fmt.Sprintf(unknownValueFormat, message)
			}
		} else {
			WARN.Printf("Unable to find default language option (%s); messages for unsupported locales will never be translated", defaultLanguageOption)
			return fmt.Sprintf(unknownValueFormat, message)
		}
	}

	// This works because unlike the goconfig documentation suggests it will actually
	// try to resolve message in DEFAULT if it did not find it in the given section.
	value, err := messageConfig.String(region, message)
	if err != nil {
		WARN.Printf("Unknown message '%s' for locale '%s'", message, locale)
		return fmt.Sprintf(unknownValueFormat, message)
	}

	if len(args) > 0 {
		TRACE.Printf("Arguments detected, formatting '%s' with %v", value, args)
		value = fmt.Sprintf(value, args...)
	}

	return value
}

func parseLocale(locale string) (language, region string) {
	if strings.Contains(locale, "-") {
		languageAndRegion := strings.Split(locale, "-")
		return languageAndRegion[0], languageAndRegion[1]
	}

	return locale, ""
}

// Retrieve message format or default format when i18n message is missing.
func getUnknownValueFormat() string {
	return Config.StringDefault(unknownFormatConfigKey, defaultUnknownFormat)
}

// Recursively read and cache all available messages from all message files on the given path.
func loadMessages(path string) {
	messages = make(map[string]*config.Config)

	// Read in messages from the modules. Load the module messges first,
	// so that it can be override in parent application
	for _, module := range Modules {
		TRACE.Println("Importing messages from module:", module.ImportPath)
		if err := Walk(filepath.Join(module.Path, messageFilesDirectory), loadMessageFile); err != nil &&
			!os.IsNotExist(err) {
			ERROR.Println("Error reading messages files from module:", err)
		}
	}

	if err := Walk(path, loadMessageFile); err != nil && !os.IsNotExist(err) {
		ERROR.Println("Error reading messages files:", err)
	}
}

// Load a single message file
func loadMessageFile(path string, info os.FileInfo, osError error) error {
	if osError != nil {
		return osError
	}
	if info.IsDir() {
		return nil
	}

	if matched, _ := regexp.MatchString(messageFilePattern, info.Name()); matched {
		var config *config.Config
		config, err := parseMessagesFile(path)
		if err != nil {
			return err
		}
		locale := parseLocaleFromFileName(info.Name())

		// If we have already parsed a message file for this locale, merge both
		if _, exists := messages[locale]; exists {
			messages[locale].Merge(config)
			TRACE.Printf("Successfully merged messages for locale '%s'", locale)
		} else {
			messages[locale] = config
		}

		TRACE.Println("Successfully loaded messages from file", info.Name())
	} else {
		TRACE.Printf("Ignoring file %s because it did not have a valid extension", info.Name())
	}

	return nil
}

func parseMessagesFile(path string) (messageConfig *config.Config, err error) {
	messageConfig, err = config.ReadDefault(path)
	return
}

func parseLocaleFromFileName(file string) string {
	extension := filepath.Ext(file)[1:]
	return strings.ToLower(extension)
}

func init() {
	OnAppStart(func() {
		loadMessages(filepath.Join(BasePath, messageFilesDirectory))
	})
}

func I18nFilter(c *Controller, fc []Filter) {
	if foundCookie, cookieValue := hasLocaleCookie(c.Request); foundCookie {
		TRACE.Printf("Found locale cookie value: %s", cookieValue)
		setCurrentLocaleControllerArguments(c, cookieValue)
	} else if foundHeader, headerValue := hasAcceptLanguageHeader(c.Request); foundHeader {
		TRACE.Printf("Found Accept-Language header value: %s", headerValue)
		setCurrentLocaleControllerArguments(c, headerValue)
	} else {
		TRACE.Println("Unable to find locale in cookie or header, using empty string")
		setCurrentLocaleControllerArguments(c, "")
	}
	fc[0](c, fc[1:])
}

// Set the current locale controller argument (CurrentLocaleControllerArg) with the given locale.
func setCurrentLocaleControllerArguments(c *Controller, locale string) {
	c.Request.Locale = locale
	c.ViewArgs[CurrentLocaleRenderArg] = locale
}

// Determine whether the given request has valid Accept-Language value.
//
// Assumes that the accept languages stored in the request are sorted according to quality, with top
// quality first in the slice.
func hasAcceptLanguageHeader(request *Request) (bool, string) {
	if request.AcceptLanguages != nil && len(request.AcceptLanguages) > 0 {
		return true, request.AcceptLanguages[0].Language
	}

	return false, ""
}

// Determine whether the given request has a valid language cookie value.
func hasLocaleCookie(request *Request) (bool, string) {
	if request != nil && request.Cookies() != nil {
		name := Config.StringDefault(localeCookieConfigKey, CookiePrefix+"_LANG")
		cookie, err := request.Cookie(name)
		if err == nil {
			return true, cookie.Value
		}
		TRACE.Printf("Unable to read locale cookie with name '%s': %s", name, err.Error())
	}

	return false, ""
}
