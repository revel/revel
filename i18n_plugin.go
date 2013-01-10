package rev

const (
	CurrentLocaleRenderArg string = "currentLocale" // The key for the current locale render arg value

	localeCookieConfigKey string = "i18n.cookie" // The config option key for the current locale cookie name
)

func init() {
	RegisterPlugin(I18nPlugin{})
}

type I18nPlugin struct {
	EmptyPlugin
}

func (p I18nPlugin) BeforeRequest(c *Controller) {
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
}

// Set the current locale controller argument (CurrentLocaleControllerArg) with the given locale.
func setCurrentLocaleControllerArguments(c *Controller, locale string) {
	c.Locale = locale
	c.RenderArgs[CurrentLocaleRenderArg] = locale
}

// Determine whether the given request has valid Accept-Language value.
//
// Assumes that the accept languages stored in the request are sorted according to quality, with top quality first in the slice.
func hasAcceptLanguageHeader(request *Request) (bool, string) {
	if request.AcceptLanguages != nil && len(request.AcceptLanguages) > 0 {
		return true, request.AcceptLanguages[0].Language
	}

	return false, ""
}

// Determine whether the given request has a valid language cookie value.
func hasLocaleCookie(request *Request) (bool, string) {
	if request != nil && request.Cookies() != nil {
		if name, found := Config.String(localeCookieConfigKey); found {
			if cookie, error := request.Cookie(name); error == nil {
				return true, cookie.Value
			} else {
				TRACE.Printf("Unable to read locale cookie with name '%s': %s", name, error.Error())
			}
		} else {
			ERROR.Printf("Unable to find configured locale cookie (%s), please correct the application configuration file!", localeCookieConfigKey)
		}
	}

	return false, ""
}
