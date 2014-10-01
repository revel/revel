package tests

import (
	"encoding/json"
	"net/url"
	"time"

	"github.com/revel/revel"
)

var personaTestUsers map[string]PersonaTestUser

type AppTest struct {
	revel.TestSuite
}

type PersonaTestUser struct {
	Assertion string `json:"assertion"`
	Audience  string `json:"audience"`
	Email     string `json:"email"`
	Pass      string `json:"pass"`
	Received  time.Time
}

func (t *AppTest) TestThatLoginPageWorks() {
	// Make sure empty assertion will cause an error.
	t.PostForm("/login", url.Values{
		"assertion": {""},
	})
	t.AssertStatus(400)

	// Ensure that incorrect audience parameter will lead to an error.
	user := t.EmailWithAssertion("https://example.com")
	t.PostForm("/login", url.Values{
		"assertion": {user.Assertion},
	})
	t.AssertEqual(user.Audience, "https://example.com")
	t.AssertStatus(400)

	// Check whether authentication works.
	user = t.EmailWithAssertion("http://" + revel.Config.StringDefault("http.host", "localhost"))
	t.PostForm("/login", url.Values{
		"assertion": {user.Assertion},
	})
	t.AssertOk()
	t.AssertContains("Login successful")

	// Make sure user is authenticated now.
	t.Get("/")
	t.AssertContains(user.Email)
}

func (t *AppTest) TestThatLogoutPageWorks() {
	// Authenticating a user.
	user := t.EmailWithAssertion("http://" + revel.Config.StringDefault("http.host", "localhost"))
	t.PostForm("/login", url.Values{
		"assertion": {user.Assertion},
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
// The test persona service expects audience to begin with protocol, for example: "http://".
func (t *AppTest) EmailWithAssertion(audience string) PersonaTestUser {
	// The process of getting new test users takes a lot of time. To reduce the number
	// of http requests using the same user data till they are up-to-date.
	if user, ok := personaTestUsers[audience]; ok {
		// Make sure user data are still valid.
		// Data expire after 2 minutes. We are updating them after 1 just in case.
		if !time.Now().After(user.Received.Add(time.Minute)) {
			return user
		}
	}

	// Trying to get data from testing server.
	u := "http://personatestuser.org"
	urn := "/email_with_assertion/" + url.QueryEscape(audience)

	req := t.GetCustom(u + urn)
	req.URL.Opaque = urn // Use unescaped version of URN for the request.
	req.Send()

	// Check whether response status is OK.
	revel.TRACE.Printf("PERSONA TESTING: Response of testing server is %q", t.ResponseBody)
	t.AssertOk()

	// Parsing the response from server.
	var user PersonaTestUser
	err := json.Unmarshal(t.ResponseBody, &user)
	t.Assert(err == nil)

	// Register the time when new user data are received. We are not using "Expire"
	// parameter from persona test server because then we'll have to synchronise our clock.
	user.Received = time.Now()

	// Cache the user data.
	personaTestUsers[audience] = user

	return user
}

func init() {
	personaTestUsers = map[string]PersonaTestUser{}
}
