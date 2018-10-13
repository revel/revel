package revel

// SessionFilter is a Revel Filter that retrieves and sets the session cookie.
// Within Revel, it is available as a Session attribute on Controller instances.
// The name of the Session cookie is set as CookiePrefix + "_SESSION".
import ()

var sessionLog = RevelLog.New("section", "session")

func SessionFilter(c *Controller, fc []Filter) {
	CurrentSessionEngine.Decode(c)
	sessionWasEmpty := c.Session.Empty()

	// Make session vars available in templates as {{.session.xyz}}
	c.ViewArgs["session"] = c.Session
	c.ViewArgs["_controller"] = c

	fc[0](c, fc[1:])

	// If session is not empty or if session was not empty then
	// pass it back to the session engine to be encoded
	if !c.Session.Empty() || !sessionWasEmpty {
		CurrentSessionEngine.Encode(c)
	}
}
