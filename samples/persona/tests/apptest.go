package tests

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/revel/revel"
)

type AppTest struct {
	revel.TestSuite
}

type PersonaTestUser struct {
	Assertion string `json:"assertion"`
	Audience  string `json:"audience"`
	Email     string `json:"email"`
	Pass      string `json:"pass"`
}

func (t AppTest) TestThatLoginPageWorks() {
	// Make sure empty assertion will cause an error.
	t.PostForm("/login", url.Values{
		"assertion": []string{""},
	})
	t.AssertStatus(400)

	// Ensure that incorrect audience parameter will lead to an error.
	user := t.EmailWithAssertion("https://example.com")
	t.PostForm("/login", url.Values{
		"assertion": []string{user.Assertion},
	})
	t.AssertEqual(user.Audience, "https://example.com")
	t.AssertStatus(400)

	// Check whether authentication works.
	user = t.EmailWithAssertion("http://" + revel.Config.StringDefault("http.host", "localhost"))
	t.PostForm("/login", url.Values{
		"assertion": []string{user.Assertion},
	})
	t.AssertOk()
	t.AssertContains("Login successful")

	// Make sure user is authenticated now.
	t.Get("/")
	t.AssertContains(user.Email)
}

func (t AppTest) TestThatLogoutPageWorks() {
	// Authenticating a user.
	user := t.EmailWithAssertion("http://" + revel.Config.StringDefault("http.host", "localhost"))
	t.PostForm("/login", url.Values{
		"assertion": []string{user.Assertion},
	})
	t.AssertOk()
	t.AssertContains("Login successful")

	// Make sure user is authenticated now.
	t.Get("/")
	t.AssertContains(user.Email)

	// Trying to sign out.
	t.Get("/logout")

	// Make sure user is not logged in.
	t.Get("/")
	t.AssertContains("Signin with your email")
}

// EmailWithAssertion uses personatestuser.org service for getting testing parameters.
// Audience is expected to begin with protocol, for example: "http://".
func (t AppTest) EmailWithAssertion(audience string) *PersonaTestUser {
	// Trying to get data from testing server.
	uri := "/email_with_assertion/" + url.QueryEscape(audience)
	req, err := http.NewRequest("GET", "http://2personatestuser.org"+uri, nil)
	t.Assert(err == nil)
	req.URL.Opaque = uri // Use unescaped version of URI for request.
	t.MakeRequest(req)

	// Check whether response status is OK.
	revel.TRACE.Printf("PERSONA TESTING: Response of testing server is %q", t.ResponseBody)
	t.AssertOk()

	// Parsing the response from server.
	var user PersonaTestUser
	err = json.Unmarshal(t.ResponseBody, &user)
	t.Assert(err == nil)

	return &user
}
