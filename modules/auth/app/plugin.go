package auth

import (
	"fmt"
	"github.com/robfig/revel"
	"net/http"
	"reflect"
)

const (
	DEFAULT_PASSWORD_FIELD = "Password"
	DEFAULT_USERNAME_FIELD = "Username"
)

// struct for passing user-defined structs for use in authentication
type AuthStructs struct {
	Session *interface{}
	User    *interface{}
}

// defines resource that needs authentication
type AuthenticatedResource struct {
	Resource interface{}
	Role     string // TODO: implement role-based ACL config
}

var (
	Structs       AuthStructs
	UseRoles      bool
	PasswordField string
	UsernameField string
)

func init() {
	PasswordField = revel.Config.
		StringDefault("auth.passwordfield", DEFAULT_PASSWORD_FIELD)
	UsernameField = revel.Config.
		StringDefault("auth.usernamefield", DEFAULT_USERNAME_FIELD)
}

// The actual filter added to the resource. It checks for valid session
// information and redirects the response to create a new session if it is not
// available or valid.
var SessionAuthenticationFilter = func(c *revel.Controller, fc []revel.Filter) {
	if Structs.User == nil {
		revel.ERROR.Fatal("User struct has not been passed.")
	}
	if !false { // TODO: check for session information
		c.Flash.Error("Session invalid. Please login.")
		c.Response.Status = http.StatusFound
		c.Response.Out.Header().Add("Location", "/session/create")
	}

	fc[0](c, fc[1:]) // Execute the next filter stage.
}

// auth.Apply is run by the developer in the init.go file for his/her project.
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

func Use(s AuthStructs) {
	// test for expected fields
	var found bool
	if _, found = reflect.TypeOf(s.User).FieldByName(PasswordField); !found {
		revel.ERROR.Fatal(fmt.Sprintf("Expecting a User struct that contains the field '%v'.", PasswordField))
	}
	if _, found = reflect.TypeOf(s.User).FieldByName(UsernameField); !found {
		revel.ERROR.Fatal(fmt.Sprintf("Expecting a User struct that contains the field '%v'.", UsernameField))
	}
	Structs = s
}
