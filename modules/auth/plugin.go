package auth

import (
	"fmt"
	"github.com/robfig/revel"
	"github.com/robfig/revel/cache"
	"net/http"
	"reflect"
	"time"
)

const (
	DEFAULT_PASSWORD_FIELD    = "Password"
	DEFAULT_PASSWORD_USE_SALT = false
	DEFAULT_USERNAME_FIELD    = "Username"
	SESSION_KEY               = "BasicAuthSessionId"
)

var (
	SessionId       string
	Structs         AuthStructs
	UseRoles        bool
	PasswordField   string
	PasswordUseSalt bool
	UsernameField   string
)

func init() {
	revel.OnAppStart(func() {
		PasswordField = revel.Config.
			StringDefault("auth.password.field", DEFAULT_PASSWORD_FIELD)
		PasswordUseSalt = revel.Config.
			BoolDefault("auth.password.usesalt", DEFAULT_PASSWORD_USE_SALT)
		UsernameField = revel.Config.
			StringDefault("auth.username.field", DEFAULT_USERNAME_FIELD)
	})
}

// The actual filter added to the resource. It checks for valid session
// information and redirects the response to create a new session if it is not
// available or valid.
var SessionAuthenticationFilter = func(c *revel.Controller, fc []revel.Filter) {
	if Structs.User == nil {
		revel.ERROR.Fatal("User struct has not been passed.")
	}
	SessionId = c.Session.Id()
	if valid := CheckSession(c); !valid {
		c.Flash.Error("Session invalid. Please login.")
		c.Response.Status = http.StatusFound
		c.Response.Out.Header().Add("Location", "/session/create")
	}

	fc[0](c, fc[1:]) // Execute the next filter stage.
}

// CheckSession is called by SessionAuthenticationFilter to check for a valid
// session.
func CheckSession(c *revel.Controller) bool {
	if value, ok := c.Session[SESSION_KEY]; ok {
		return VerifySession(value)
	}
	return false
}

// VerifySession checks stored session id against stored value
func VerifySession(sid string) bool {
	var session Session
	if err := cache.Get(SessionId+SESSION_KEY, &session); err != nil {
		return false
	}
	return sid == session.Id
}

func InvalidateSession() {
	go cache.Delete(SessionId + SESSION_KEY)
}

// Apply is run by the developer in the init.go file for his/her project.
// It loops over the slice for all AuthenticatedResources the developer wishes
// to be protected with authentication.
func Apply(m []AuthenticatedResource) {
	for _, a := range m {
		var fc revel.FilterConfigurator
		if reflect.TypeOf(a.Resource).Kind() == reflect.Func {
			fc = revel.FilterAction(a.Resource)
		} else {
			fc = revel.FilterController(a.Resource)
		}
		fc.Add(SessionAuthenticationFilter)
	}
}

// Use is run by the developer in the init.go file for his/her project.
// It should contain references the the structs used to contain user and,
// optionally, session information. It also verifies the structs contain the
// expected fields, whether the defaults or those defined in the app.conf.
//
// Example:
//     import (
//         "github.com/robfig/revel/auth"
//         "project/models"
//     )
//
//     func init() {
//         revel.OnAppStart(func() {
//	           auth.Use(auth.AuthStructs{
//	               Session: models.Session{},
//                 User:    models.User{},
//             })
//         })
//     }
func Use(s AuthStructs) {
	var found bool
	if _, found = reflect.TypeOf(s.User).FieldByName(PasswordField); !found {
		revel.ERROR.Fatal(fmt.Sprintf(
			"Expecting a User struct that contains the field '%v'.",
			PasswordField))
	}
	if _, found = reflect.TypeOf(s.User).FieldByName(UsernameField); !found {
		revel.ERROR.Fatal(fmt.Sprintf(
			"Expecting a User struct that contains the field '%v'.",
			UsernameField))
	}
	Structs = s
}

// struct for passing user-defined structs for use in authentication
type AuthStructs struct {
	// Session interface{} // TODO: do this. let's start with cache storage
	User interface{}
}

// defines resource that needs authentication
type AuthenticatedResource struct {
	Resource interface{}
	Role     string // TODO: allow role-based ACL config
}

type Session struct {
	Id        string
	Data      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
