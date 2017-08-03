package logger

import "gopkg.in/inconshreveable/log15.v2/stack"

// For logging purposes the call stack can be used to record the stack trace of a bad error
// simply pass it as a context field in your log statement like
// `controller.Log.Critc("This should not occur","stack",revel.NewCallStack())`
func NewCallStack() interface{} {
	return stack.Callers()
}
