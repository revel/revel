package revel

// The session engine provides an interface to allow for storage of session data
type (
	SessionEngine interface {
		Decode(c *Controller) // Called to decode the session information on the controller
		Encode(c *Controller) // Called to encode the session information on the controller
	}
)

var (
	sessionEngineMap     = map[string]func() SessionEngine{}
	CurrentSessionEngine SessionEngine
)

// Initialize session engine on startup
func init() {
	OnAppStart(initSessionEngine, 5)
}

func RegisterSessionEngine(f func() SessionEngine, name string) {
	sessionEngineMap[name] = f
}

// Called when application is starting up
func initSessionEngine() {
	// Check for session engine to use and assign it
	sename := Config.StringDefault("session.engine", "revel-cookie")
	if se, found := sessionEngineMap[sename]; found {
		CurrentSessionEngine = se()
	} else {
		sessionLog.Warn("Session engine '%s' not found, using default session engine revel-cookie", sename)
		CurrentSessionEngine = sessionEngineMap["revel-cookie"]()
	}
}
