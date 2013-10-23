---
title: Facebook OAuth2
layout: samples
---

The facebook-oauth2 app demonstrates:

* Using the `goauth2` library to fetch json information on the logged-in
  Facebook user.

Here are the contents of the app:

	facebook-oauth2/app/
		models
			user.go   # User struct and in-memory data store
		controllers
			app.go    # All code

[Browse the code on Github](https://github.com/robfig/revel/tree/master/samples/facebook-oauth2)

## OAuth2 Overview

The entire OAuth process is governed by this configuration:

<pre class="prettyprint lang-go">{% capture guy %}{% raw %}
var FACEBOOK = &oauth.Config{
	ClientId:     "95341411595",
	ClientSecret: "8eff1b488da7fe3426f9ecaf8de1ba54",
	AuthURL:      "https://graph.facebook.com/oauth/authorize",
	TokenURL:     "https://graph.facebook.com/oauth/access_token",
	RedirectURL:  "http://loisant.org:9000/Application/Auth",
}{% endraw %}{% endcapture %}{{ guy|escape }}
</pre>

Here's an overview of the process:

1. The app sends the user to **AuthURL**.
2. While there, the user agrees to the authorization.
3. Facebook sends the user back to **RedirectURL**, adding a parameter **code**.
4. The app retrieves an OAuth access token from **TokenURL** using the **code**.
5. The app subsequently uses the access token to authenticate web service requests.

## Code walk

Let's take a look at the first bit of code:

<pre class="prettyprint lang-go">{% capture guy %}{% raw %}
func (c Application) Index() revel.Result {
	u := c.connected()
	me := map[string]interface{}{}
	if u != nil && u.AccessToken != "" {
		// Use the access token to request user info
		...
	}

	authUrl := FACEBOOK.AuthCodeURL("foo")
	return c.Render(me, authUrl)
}{% endraw %}{% endcapture %}{{ guy|escape }}
</pre>

It doesn't do much since we don't have an access token yet.  All it does is
generate an Authorization URL.  ("foo" is the "state", which is a parameter that
facebook propagates back to us as a parameter to the RedirectURL.  We do not
need to use it here.)

Here's the interesting bit of the template:

{% raw %}
	{{if .me}}
	<h3>You're {{.me.name}} on Facebook</h3>
	{{else}}
	<a href="{{.authUrl}}">login</a>
	{{end}}
{% endraw %}

If we had information on the user, we would tell them their name.  Since we
don't, we just ask the user to log in to Facebook.

Assuming the user does so, the next time we see them is when Facebook sends them
to `Auth`:

<pre class="prettyprint lang-go">{% capture guy %}{% raw %}
func (c Application) Auth(code string) revel.Result {
	t := &oauth.Transport{Config: FACEBOOK}
	tok, err := t.Exchange(code)
	if err != nil {
		revel.ERROR.Println(err)
		return c.Redirect(Application.Index)
	}

	user := c.connected()
	user.AccessToken = tok.AccessToken
	return c.Redirect(Application.Index)
}{% endraw %}{% endcapture %}{{ guy|escape }}
</pre>

The `t.Exchange(code)` bit makes a request to the **TokenURL** to get the access
token.  If successful, we store it on the user.  Either way, the user ends up
back at `Index`:

<pre class="prettyprint lang-go">{% capture guy %}{% raw %}
func (c Application) Index() revel.Result {
	u := c.connected()
	me := map[string]interface{}{}
	if u != nil && u.AccessToken != "" {
		resp, _ := http.Get("https://graph.facebook.com/me?access_token=" +
			url.QueryEscape(u.AccessToken))
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
			revel.ERROR.Println(err)
		}
{% endraw %}{% endcapture %}{{ guy|escape }}
</pre>

Now we have an AccessToken, so we make a request to get the associated user's
information.  The information gets returned in JSON, so we decode it into a
simple map and pass it into the template.
