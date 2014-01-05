---
title: Session / Flash Scopes
layout: manual
---

Revel provides two cookie-based storage mechanisms.

<pre class="prettyprint lang-go">
// A signed cookie (and thus limited to 4kb in size).
// Restriction: Keys may not have a colon in them.
type Session map[string]string

// Flash represents a cookie that gets overwritten on each request.
// It allows data to be stored across one page at a time.
// This is commonly used to implement success or error messages.
// e.g. the Post/Redirect/Get pattern: http://en.wikipedia.org/wiki/Post/Redirect/Get
type Flash struct {
	Data, Out map[string]string
}
</pre>

## Session

Revel's concept of "session" is a string map, stored as a cryptographically
signed cookie.

This has a couple implications:
* The size limit is 4kb.
* All data must be serialized to a string for storage.
* All data may be viewed by the user (it is not encrypted), but it is safe from modification.

The default lifetime of the session cookie is the browser lifetime.  This
can be overriden to a specific amount of time by setting the session.expires
option in app.config.  The format is that of
[time.ParseDuration](http://golang.org/pkg/time/#ParseDuration).

## Flash

The Flash provides single-use string storage. It useful for implementing
[the Post/Redirect/Get pattern](http://en.wikipedia.org/wiki/Post/Redirect/Get),
or for transient "Operation Successful!" or "Operation Failed!" messages.

Here's an example of that pattern:

<pre class="prettyprint lang-go">
// Show the Settings form
func (c App) ShowSettings() revel.Result {
	return c.Render()
}

// Process a post
func (c App) SaveSettings(setting string) revel.Result {
    // Make sure `setting` is provided and not empty
    c.Validation.Required(setting)
    if c.Validation.HasErrors() {
        // Sets the flash parameter `error` which will be sent by a flash cookie
        c.Flash.Error("Settings invalid!")
        // Keep the validation error from above by setting a flash cookie
        c.Validation.Keep()
        // Copies all given parameters (URL, Form, Multipart) to the flash cookie
        c.FlashParams()
        return c.Redirect(App.ShowSettings)
    }
    saveSetting(setting)
    // Sets the flash cookie to contain a success string
    c.Flash.Success("Settings saved!")
    return c.Redirect(App.ShowSettings)
}
</pre>

Walking through this example:
1. User fetches the settings page.
2. User posts a setting (POST)
3. Application processes the request, saves an error or success message to the flash, and redirects the user to the settings page (REDIRECT)
4. User fetches the settings page, whose template shows the flashed message. (GET)

It uses two convenience functions:
1. `Flash.Success(message string)` is an abbreviation of `Flash.Out["success"] = message`
2. `Flash.Error(message string)` is an abbreviation of `Flash.Out["error"] = message`

Flash messages may be referenced by key in templates.  For example, to access
the success and error messages set by the convenience functions, use these
expressions:

{% raw %}
	{{.flash.success}}
	{{.flash.error}}
{% endraw %}
