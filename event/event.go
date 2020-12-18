package event

type AsyncEvent interface {
	Async() bool
}

// Listener is a event listener
type Listener interface{}

// Store is a interface for event store
type Store interface {
	Listen(eventName string, listener Listener)
	Publish(eventName string, evt interface{})
	SetManager(manager Manager)
}

type Manager interface {
	Listen(listeners ...Listener)
	Publish(evt interface{})
	Call(evt interface{}, listener Listener)
}
