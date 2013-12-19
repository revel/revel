package controllers

import (
	"encoding/json"
	"fmt"
	"code.google.com/p/goauth2/oauth"
	"github.com/robfig/revel"
	"github.com/robfig/revel/samples/facebook-oauth2/app/models"
	"net/http"
	"net/url"
	"strconv"
)

type Application struct {
	*revel.Controller
}

// The following keys correspond to a test application
// registered on Facebook, and associated with the loisant.org domain.
// You need to bind loisant.org to your machine with /etc/hosts to
// test the application locally.

var FACEBOOK = &oauth.Config{
	ClientId:     "95341411595",
	ClientSecret: "8eff1b488da7fe3426f9ecaf8de1ba54",
	AuthURL:      "https://graph.facebook.com/oauth/authorize",
	TokenURL:     "https://graph.facebook.com/oauth/access_token",
	RedirectURL:  "http://loisant.org:9000/Application/Auth",
}

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
		revel.INFO.Println(me)
	}

	authUrl := FACEBOOK.AuthCodeURL("foo")
	return c.Render(me, authUrl)
}

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
}

func setuser(c *revel.Controller) revel.Result {
	var user *models.User
	if _, ok := c.Session["uid"]; ok {
		uid, _ := strconv.ParseInt(c.Session["uid"], 10, 0)
		user = models.GetUser(int(uid))
	}
	if user == nil {
		user = models.NewUser()
		c.Session["uid"] = fmt.Sprintf("%d", user.Uid)
	}
	c.RenderArgs["user"] = user
	return nil
}

func init() {
	revel.InterceptFunc(setuser, revel.BEFORE, &Application{})
}

func (c Application) connected() *models.User {
	return c.RenderArgs["user"].(*models.User)
}
