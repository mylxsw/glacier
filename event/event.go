package event

import "context"

type AsyncEvent interface {
	Async() bool
}

// Store is an interface for event store
type Store interface {
	Listen(eventName string, listener interface{})
	Publish(evt Event) error
	SetManager(manager Manager)
	Start(ctx context.Context) <-chan interface{}
}

type Event struct {
	Name  string
	Event interface{}
}

type Manager interface {
	Publisher
	Listener
	Call(evt interface{}, listener interface{})
	Start(ctx context.Context) <-chan interface{}
}

type Publisher interface {
	Publish(evt interface{}) error
}

type Listener interface {
	Listen(listeners ...interface{})
}
