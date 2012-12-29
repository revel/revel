package rev

import (
	"fmt"
	"github.com/robfig/goconfig/config"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	messageFilesDirectory string = "messages"
	messageFilePattern    string = `^\w+.[a-zA-Z]{2}$`
	unknownValueFormat    string = "??? %s ???"
	defaultLanguageOption string = "i18n.default_language"
)

var (
	// All currently loaded message configs.
	messages map[string]*config.Config
)

// Return all currently loaded message languages.
func MessageLanguages() []string {
	languages := make([]string, len(messages))
	i := 0
	for language, _ := range messages {
		languages[i] = language
		i++
	}
	return languages
}

// Perform a message look-up for the given locale and message using the given arguments.
//
// When either an unknown locale or message is detected, a specially formatted string is returned.
func Message(locale, message string, args ...interface{}) (value string) {
	language, region := parseLocale(locale)
	TRACE.Printf("Resolving message '%s' for language '%s' and region '%s'", message, language, region)

	messageConfig, knownLanguage := messages[language]
	if !knownLanguage {
		WARN.Printf("Unsupported language for locale '%s' and message '%s', trying default language", locale, message)

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
	value, error := messageConfig.String(region, message)
	if error != nil {
		WARN.Printf("Unknown message '%s' for locale '%s'", message, locale)
		return fmt.Sprintf(unknownValueFormat, message)
	}

	if args != nil && len(args) > 0 {
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

func init() {
	InitHooks = append(InitHooks, loadMessagesOnInitialize)
}

// Wrap loadMessages() in a function so it can be hooked into InitHooks.
func loadMessagesOnInitialize() {
	loadMessages(path.Join(BasePath, messageFilesDirectory))
}

// Recursively read and cache all available messages from all message files on the given path.
func loadMessages(path string) {
	messages = make(map[string]*config.Config)

	if error := filepath.Walk(path, loadMessageFile); error != nil {
		ERROR.Println("Error reading messages files:", error)
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
		if config, error := parseMessagesFile(path); error != nil {
			return error
		} else {
			locale := parseLocaleFromFileName(info.Name())

			// If we have already parsed a message file for this locale, merge both
			if _, exists := messages[locale]; exists {
				messages[locale].Merge(config)
				TRACE.Printf("Successfully merged messages for locale '%s'", locale)
			} else {
				messages[locale] = config
			}

			TRACE.Println("Successfully loaded messages from file", info.Name())
		}
	} else {
		TRACE.Printf("Ignoring file %s because it did not have a valid extension", info.Name())
	}

	return nil
}

func parseMessagesFile(path string) (messageConfig *config.Config, error error) {
	messageConfig, error = config.ReadDefault(path)
	return
}

func parseLocaleFromFileName(file string) string {
	extension := filepath.Ext(file)[1:]
	return strings.ToLower(extension)
}
