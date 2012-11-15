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
)

var (
	// All currently loaded locales.
	locales []string

	// All currently loaded message configs.
	messages map[string]*config.Config
)

// Return all currently loaded message locales.
func MessageLocales() []string {
	return locales
}

// Perform a message look-up for the given locale and message using the given arguments.
//
// When either an unknown locale or message is detected, a specially formatted string is returned.
func Message(locale, message string, args ...interface{}) (value string) {
	language, region := parseLocale(locale)

	messageConfig, knownLocale := messages[language]
	if !knownLocale {
		WARN.Printf("Unknown locale '%s' for message '%s'", locale, message)
		return fmt.Sprintf(unknownValueFormat, locale)
	}

	// This works because unlike the goconfig documentation suggests it will actually
	// try to resolve message in DEFAULT if it did not find it in the given section.
	value, error := messageConfig.String(region, message)
	if error != nil {
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
			locales = append(locales, locale)

			if _, exists := messages[locale]; exists {
				// TODO: fix this so that messages from identical locale files are merged
				ERROR.Println("Multiple messages files for locale", locale)
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
