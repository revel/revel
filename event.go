package revel

type (
	// The event type
	Event int
	// The event response
	EventResponse int
	// The handler signature
	EventHandler func(typeOf Event, value interface{}) (responseOf EventResponse)
)

const (
	// Event type when templates are going to be refreshed (receivers are registered template engines added to the template.engine conf option)
	TEMPLATE_REFRESH_REQUESTED Event = iota
	// Event type when templates are refreshed (receivers are registered template engines added to the template.engine conf option)
	TEMPLATE_REFRESH_COMPLETED
	// Event type before all module loads, events thrown to handlers added to AddInitEventHandler

	// Event type before all module loads, events thrown to handlers added to AddInitEventHandler
	REVEL_BEFORE_MODULES_LOADED
	// Event type after all module loads, events thrown to handlers added to AddInitEventHandler
	REVEL_AFTER_MODULES_LOADED

	// Event type before server engine is initialized, receivers are active server engine and handlers added to AddInitEventHandler
	ENGINE_BEFORE_INITIALIZED
	// Event type before server engine is started, receivers are active server engine and handlers added to AddInitEventHandler
	ENGINE_STARTED

	// Event raised when the engine is told to shutdown
	ENGINE_SHUTDOWN_REQUEST

	// Event type after server engine is stopped, receivers are active server engine and handlers added to AddInitEventHandler
	ENGINE_SHUTDOWN

	// Called before routes are refreshed
	ROUTE_REFRESH_REQUESTED
	// Called after routes have been refreshed
	ROUTE_REFRESH_COMPLETED

	// Fired when a panic is caught during the startup process
	REVEL_FAILURE
)

// Fires system events from revel
func RaiseEvent(key Event, value interface{}) (response EventResponse) {
	utilLog.Info("Raising event", "len", len(initEventList))
	for _, handler := range initEventList {
		response |= handler(key, value)
	}
	return
}

// Add event handler to listen for all system events
func AddInitEventHandler(handler EventHandler) {
	initEventList = append(initEventList, handler)
	return
}
