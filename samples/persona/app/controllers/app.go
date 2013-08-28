package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/robfig/revel"
)

const host = "" // set this to your host

type App struct {
	*revel.Controller
}

type PersonaResponse struct {
	Status   string `json:"status"`
	Email    string `json:"email"`
	Audience string `json:"audience"`
	Expires  int64  `json:"expires"`
	Issuer   string `json:"issuer"`
}

type LoginResult struct {
	StatusCode int
	Message    string
}

func (r LoginResult) Apply(req *revel.Request, resp *revel.Response) {
	resp.WriteHeader(r.StatusCode, "text/html")
	resp.Out.Write([]byte(r.Message))
}

func (c App) Index() revel.Result {
	email := c.Session["email"]
	return c.Render(email)
}

func (c App) Login(assertion string) revel.Result {
	assertion = strings.TrimSpace(assertion)
	if assertion == "" {
		return &LoginResult{
			StatusCode: http.StatusBadRequest,
			Message:    "Assertion required.",
		}
	}

	values := url.Values{"assertion": {assertion}, "audience": {host}}
	resp, err := http.PostForm("https://verifier.login.persona.org/verify", values)
	if err != nil {
		return &LoginResult{
			StatusCode: http.StatusBadRequest,
			Message:    "Authentication failed.",
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &LoginResult{
			StatusCode: http.StatusBadRequest,
			Message:    "Authentication failed.",
		}
	}

	p := &PersonaResponse{}
	err = json.Unmarshal(body, p)
	if err != nil {
		return &LoginResult{
			StatusCode: http.StatusBadRequest,
			Message:    "Authentication failed.",
		}
	}

	c.Session["email"] = p.Email
	fmt.Println("Login successful: ", p.Email)

	return &LoginResult{
		StatusCode: http.StatusOK,
		Message:    "Login successful.",
	}
}

func (c App) Logout() revel.Result {
	delete(c.Session, "email")
	return c.Redirect("/")
}
