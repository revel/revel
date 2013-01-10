package rev

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

const (
	testConfigPath string = "testdata/i18n/config"
	testConfigName string = "test_app.conf"
)

func TestHasLocaleCookie(t *testing.T) {
	stubI18nConfig(t)

	if found, value := hasLocaleCookie(buildRequestWithCookie("REVEL_LANG", "en")); !found {
		t.Errorf("Expected %s cookie with value '%s' but found nothing or unexpected value '%s'", "REVEL_LANG", "en", value)
	}
	if found, value := hasLocaleCookie(buildRequestWithCookie("REVEL_LANG", "en-US")); !found {
		t.Errorf("Expected %s cookie with value '%s' but found nothing or unexpected value '%s'", "REVEL_LANG", "en-US", value)
	}
	if found, _ := hasLocaleCookie(buildRequestWithCookie("DOESNT_EXIST", "en-US")); found {
		t.Errorf("Expected %s cookie to not exist, but apparently it does", "DOESNT_EXIST")
	}
}

func TestHasLocaleCookieWithInvalidConfig(t *testing.T) {
	stubI18nConfigWithoutLanguageCookieOption(t)
	if found, _ := hasLocaleCookie(buildRequestWithCookie("REVEL_LANG", "en-US")); found {
		t.Errorf("Expected %s cookie to not exist because the configured name is missing", "REVEL_LANG")
	}
}

func TestHasAcceptLanguageHeader(t *testing.T) {
	if found, value := hasAcceptLanguageHeader(buildRequestWithAcceptLanguages("en-US")); !found && value != "en-US" {
		t.Errorf("Expected to find Accept-Language header with value '%s', found '%s' instead", "en-US", value)
	}
	if found, value := hasAcceptLanguageHeader(buildRequestWithAcceptLanguages("en-GB", "en-US", "nl")); !found && value != "en-GB" {
		t.Errorf("Expected to find Accept-Language header with value '%s', found '%s' instead", "en-GB", value)
	}
}

func TestBeforeRequest(t *testing.T) {
	stubI18nConfig(t)
	plugin := I18nPlugin{}

	controller := NewController(buildEmptyRequest(), nil, &ControllerType{reflect.TypeOf(Controller{}), nil})
	if plugin.BeforeRequest(controller); controller.Locale != "" {
		t.Errorf("Expected to find current language '%s' in controller, found '%s' instead", "", controller.Locale)
	}

	controller = NewController(buildRequestWithCookie("REVEL_LANG", "en-US"), nil, &ControllerType{reflect.TypeOf(Controller{}), nil})
	if plugin.BeforeRequest(controller); controller.Locale != "en-US" {
		t.Errorf("Expected to find current language '%s' in controller, found '%s' instead", "en-US", controller.Locale)
	}

	controller = NewController(buildRequestWithAcceptLanguages("en-GB", "en-US"), nil, &ControllerType{reflect.TypeOf(Controller{}), nil})
	if plugin.BeforeRequest(controller); controller.Locale != "en-GB" {
		t.Errorf("Expected to find current language '%s' in controller, found '%s' instead", "en-GB", controller.Locale)
	}
}

func stubI18nConfig(t *testing.T) {
	ConfPaths = append(ConfPaths, testConfigPath)
	testConfig, error := LoadConfig(testConfigName)
	if error != nil {
		t.Fatalf("Unable to load test config '%s': %s", testConfigName, error.Error())
	}
	Config = testConfig
}

func stubI18nConfigWithoutLanguageCookieOption(t *testing.T) {
	stubI18nConfig(t)
	Config.config.RemoveOption("DEFAULT", "i18n.cookie")
}

func buildRequestWithCookie(name, value string) *Request {
	httpRequest, _ := http.NewRequest("GET", "/", nil)
	request := NewRequest(httpRequest)
	request.AddCookie(&http.Cookie{name, value, "", "", time.Now(), "", 0, false, false, "", nil})
	return request
}

func buildRequestWithAcceptLanguages(acceptLanguages ...string) *Request {
	httpRequest, _ := http.NewRequest("GET", "/", nil)
	request := NewRequest(httpRequest)
	for _, acceptLanguage := range acceptLanguages {
		request.AcceptLanguages = append(request.AcceptLanguages, AcceptLanguage{acceptLanguage, 1})
	}
	return request
}

func buildEmptyRequest() *Request {
	httpRequest, _ := http.NewRequest("GET", "/", nil)
	request := NewRequest(httpRequest)
	return request
}
