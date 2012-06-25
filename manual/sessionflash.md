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

## Flash

The Flash provides single-use string storage. It useful for implementing
[the Post/Redirect/Get pattern](http://en.wikipedia.org/wiki/Post/Redirect/Get),
or for transient "Operation Successful!" or "Operation Failed!" messages.

Here's an example of that pattern:

<pre class="prettyprint lang-go">
// Show the Settings form
func (c App) ShowSettings() rev.Result {
	return c.Render()
}

// Process a post
func (c App) SaveSettings(setting string) rev.Result {
	c.Validation.Required(setting)
	if c.Validation.HasErrors() {
		c.Flash.Error("Settings invalid!")
		c.Validation.Keep()
		c.Params.Flash()
		return c.Redirect(App.ShowSettings)
	}

	saveSetting(setting)
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
1. `Flash.Success(message string)` is an abbreviation of Flash.Out\["success"] = message
2. `Flash.Error(message string)` is an abbreviation of Flash.Out\["error"] = message
