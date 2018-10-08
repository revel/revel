package revel_test

import (
	"github.com/revel/revel"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Test that the event handler can be attached and it dispatches the event received
func TestEventHandler(t *testing.T) {
	counter := 0
	newListener := func(typeOf revel.Event, value interface{}) (responseOf revel.EventResponse) {
		if typeOf == revel.REVEL_FAILURE {
			counter++
		}
		return
	}
	// Attach the same handlder twice so we expect to see the response twice as well
	revel.AddInitEventHandler(newListener)
	revel.AddInitEventHandler(newListener)
	revel.RaiseEvent(revel.REVEL_AFTER_MODULES_LOADED, nil)
	revel.RaiseEvent(revel.REVEL_FAILURE, nil)
	assert.Equal(t, counter, 2, "Expected event handler to have been called")
}
