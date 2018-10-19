// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package revel

import (
	"html/template"
	"net/http"
	"testing"
	"time"

	"github.com/revel/config"
	"github.com/revel/revel/logger"
)

const (
	testDataPath   string = "testdata/i18n"
	testConfigPath string = "testdata/i18n/config"
	testConfigName string = "test_app.conf"
)

func TestI18nLoadMessages(t *testing.T) {
	loadMessages(testDataPath)

	// Assert that we have the expected number of languages
	if len(MessageLanguages()) != 2 {
		t.Fatalf("Expected messages to contain no more or less than 2 languages, instead there are %d languages", len(MessageLanguages()))
	}
}

func TestI18nMessage(t *testing.T) {
	loadMessages(testDataPath)
	loadTestI18nConfig(t)

	// Assert that we can get a message and we get the expected return value
	if message := Message("nl", "greeting"); message != "Hallo" {
		t.Errorf("Message 'greeting' for locale 'nl' (%s) does not have the expected value", message)
	}
	if message := Message("en", "greeting"); message != "Hello" {
		t.Errorf("Message 'greeting' for locale 'en' (%s) does not have the expected value", message)
	}
	if message := Message("en", "folded"); message != "Greeting is 'Hello'" {
		t.Error("Unexpected unfolded message: ", message)
	}
	if message := Message("en", "arguments.string", "Vincent Hanna"); message != "My name is Vincent Hanna" {
		t.Errorf("Message 'arguments.string' for locale 'en' (%s) does not have the expected value", message)
	}
	if message := Message("en", "arguments.hex", 1234567, 1234567); message != "The number 1234567 in hexadecimal notation would be 12d687" {
		t.Errorf("Message 'arguments.hex' for locale 'en' (%s) does not have the expected value", message)
	}
	if message := Message("en", "folded.arguments", 12345); message != "Rob is 12345 years old" {
		t.Errorf("Message 'folded.arguments' for locale 'en' (%s) does not have the expected value", message)
	}
	if message := Message("en-AU", "greeting"); message != "G'day" {
		t.Errorf("Message 'greeting' for locale 'en-AU' (%s) does not have the expected value", message)
	}
	if message := Message("en-AU", "only_exists_in_default"); message != "Default" {
		t.Errorf("Message 'only_exists_in_default' for locale 'en-AU' (%s) does not have the expected value", message)
	}

	// Assert that message merging works
	if message := Message("en", "greeting2"); message != "Yo!" {
		t.Errorf("Message 'greeting2' for locale 'en' (%s) does not have the expected value", message)
	}

	// Assert that we get the expected return value for a locale that doesn't exist
	if message := Message("unknown locale", "message"); message != "??? message ???" {
		t.Error("Locale 'unknown locale' is not supposed to exist")
	}
	// Assert that we get the expected return value for a message that doesn't exist
	if message := Message("nl", "unknown message"); message != "??? unknown message ???" {
		t.Error("Message 'unknown message' is not supposed to exist")
	}
	// XSS
	if message := Message("en", "arguments.string", "<img src=a onerror=alert(1) />"); message != "My name is &lt;img src=a onerror=alert(1) /&gt;" {
		t.Error("XSS protection for messages is broken:", message)
	}
	// Avoid escaping HTML
	if message := Message("en", "arguments.string", template.HTML("<img src=a onerror=alert(1) />")); message != "My name is <img src=a onerror=alert(1) />" {
		t.Error("Passing safe HTML to message is broken:", message)
	}
}

func TestI18nMessageWithDefaultLocale(t *testing.T) {
	loadMessages(testDataPath)
	loadTestI18nConfig(t)

	if message := Message("doesn't exist", "greeting"); message != "Hello" {
		t.Errorf("Expected message '%s' for unknown locale to be default '%s' but was '%s'", "greeting", "Hello", message)
	}
	if message := Message("doesn't exist", "unknown message"); message != "??? unknown message ???" {
		t.Error("Message 'unknown message' is not supposed to exist in the default language")
	}
}

func TestHasLocaleCookie(t *testing.T) {
	loadTestI18nConfig(t)

	if found, value := hasLocaleCookie(buildRequestWithCookie("APP_LANG", "en").Request); !found {
		t.Errorf("Expected %s cookie with value '%s' but found nothing or unexpected value '%s'", "APP_LANG", "en", value)
	}
	if found, value := hasLocaleCookie(buildRequestWithCookie("APP_LANG", "en-US").Request); !found {
		t.Errorf("Expected %s cookie with value '%s' but found nothing or unexpected value '%s'", "APP_LANG", "en-US", value)
	}
	if found, _ := hasLocaleCookie(buildRequestWithCookie("DOESNT_EXIST", "en-US").Request); found {
		t.Errorf("Expected %s cookie to not exist, but apparently it does", "DOESNT_EXIST")
	}
}

func TestHasLocaleCookieWithInvalidConfig(t *testing.T) {
	loadTestI18nConfigWithoutLanguageCookieOption(t)
	if found, _ := hasLocaleCookie(buildRequestWithCookie("APP_LANG", "en-US").Request); found {
		t.Errorf("Expected %s cookie to not exist because the configured name is missing", "APP_LANG")
	}
	if found, _ := hasLocaleCookie(buildRequestWithCookie("REVEL_LANG", "en-US").Request); !found {
		t.Errorf("Expected %s cookie to exist", "REVEL_LANG")
	}
}

func TestHasAcceptLanguageHeader(t *testing.T) {
	if found, value := hasAcceptLanguageHeader(buildRequestWithAcceptLanguages("en-US").Request); !found && value != "en-US" {
		t.Errorf("Expected to find Accept-Language header with value '%s', found '%s' instead", "en-US", value)
	}
	if found, value := hasAcceptLanguageHeader(buildRequestWithAcceptLanguages("en-GB", "en-US", "nl").Request); !found && value != "en-GB" {
		t.Errorf("Expected to find Accept-Language header with value '%s', found '%s' instead", "en-GB", value)
	}
}

func TestBeforeRequest(t *testing.T) {
	loadTestI18nConfig(t)

	c := buildEmptyRequest()
	if I18nFilter(c, NilChain); c.Request.Locale != "" {
		t.Errorf("Expected to find current language '%s' in controller, found '%s' instead", "", c.Request.Locale)
	}

	c = buildRequestWithCookie("APP_LANG", "en-US")
	if I18nFilter(c, NilChain); c.Request.Locale != "en-US" {
		t.Errorf("Expected to find current language '%s' in controller, found '%s' instead", "en-US", c.Request.Locale)
	}

	c = buildRequestWithAcceptLanguages("en-GB", "en-US")
	if I18nFilter(c, NilChain); c.Request.Locale != "en-GB" {
		t.Errorf("Expected to find current language '%s' in controller, found '%s' instead", "en-GB", c.Request.Locale)
	}
}

func TestI18nMessageUnknownValueFormat(t *testing.T) {
	loadMessages(testDataPath)
	loadTestI18nConfigWithUnknowFormatOption(t)

	// Assert that we can get a message and we get the expected return value
	if message := Message("en", "greeting"); message != "Hello" {
		t.Errorf("Message 'greeting' for locale 'en' (%s) does not have the expected value", message)
	}

	// Assert that we get the expected return value with original format
	if message := Message("unknown locale", "message"); message != "*** message ***" {
		t.Error("Locale 'unknown locale' is not supposed to exist")
	}
	if message := Message("nl", "unknown message"); message != "*** unknown message ***" {
		t.Error("Message 'unknown message' is not supposed to exist")
	}
}

func BenchmarkI18nLoadMessages(b *testing.B) {
	excludeFromTimer(b, func() {
		RevelLog.SetHandler(logger.FuncHandler(func(r *logger.Record) error {
			return nil
		}))
	})

	for i := 0; i < b.N; i++ {
		loadMessages(testDataPath)
	}
}

func BenchmarkI18nMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Message("nl", "greeting")
	}
}

func BenchmarkI18nMessageWithArguments(b *testing.B) {
	excludeFromTimer(b, func() {
		RevelLog.SetHandler(logger.FuncHandler(func(r *logger.Record) error {
			return nil
		}))
	})


	for i := 0; i < b.N; i++ {
		Message("en", "arguments.string", "Vincent Hanna")
	}
}

func BenchmarkI18nMessageWithFoldingAndArguments(b *testing.B) {
	excludeFromTimer(b, func() {
		RevelLog.SetHandler(logger.FuncHandler(func(r *logger.Record) error {
			return nil
		}))
	})


	for i := 0; i < b.N; i++ {
		Message("en", "folded.arguments", 12345)
	}
}

// Exclude whatever operations the given function performs from the benchmark timer.
func excludeFromTimer(b *testing.B, f func()) {
	b.StopTimer()
	f()
	b.StartTimer()
}

func loadTestI18nConfig(t *testing.T) {
	ConfPaths = append(ConfPaths, testConfigPath)
	testConfig, err := config.LoadContext(testConfigName, ConfPaths)
	if err != nil {
		t.Fatalf("Unable to load test config '%s': %s", testConfigName, err.Error())
	}
	Config = testConfig
	CookiePrefix = Config.StringDefault("cookie.prefix", "REVEL")
}

func loadTestI18nConfigWithoutLanguageCookieOption(t *testing.T) {
	loadTestI18nConfig(t)
	Config.Raw().RemoveOption("DEFAULT", "i18n.cookie")
}

func loadTestI18nConfigWithUnknowFormatOption(t *testing.T) {
	loadTestI18nConfig(t)
	Config.Raw().AddOption("DEFAULT", "i18n.unknown_format", "*** %s ***")
}

func buildRequestWithCookie(name, value string) *Controller {
	httpRequest, _ := http.NewRequest("GET", "/", nil)
	controller := NewTestController(nil, httpRequest)

	httpRequest.AddCookie(&http.Cookie{
		Name:       name,
		Value:      value,
		Path:       "",
		Domain:     "",
		Expires:    time.Now(),
		RawExpires: "",
		MaxAge:     0,
		Secure:     false,
		HttpOnly:   false,
		Raw:        "",
		Unparsed:   nil,
	})
	return controller
}

func buildRequestWithAcceptLanguages(acceptLanguages ...string) *Controller {
	httpRequest, _ := http.NewRequest("GET", "/", nil)
	controller := NewTestController(nil, httpRequest)

	request := controller.Request
	for _, acceptLanguage := range acceptLanguages {
		request.AcceptLanguages = append(request.AcceptLanguages, AcceptLanguage{acceptLanguage, 1})
	}
	return controller
}

func buildEmptyRequest() *Controller {
	httpRequest, _ := http.NewRequest("GET", "/", nil)
	controller := NewTestController(nil, httpRequest)
	return controller
}
