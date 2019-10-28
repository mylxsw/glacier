package event

import (
	"fmt"
	"reflect"
	"sync"
)

type Manager interface {
	Listen(listeners ...Listener)
	Publish(evt interface{})
	Call(evt interface{}, listener Listener)
}

// Listener is a event listener
type Listener interface{}

// Store is a interface for event store
type Store interface {
	Listen(eventName string, listener Listener)
	Publish(eventName string, evt interface{})
	SetManager(manager Manager)
}

// eventManager is a manager for event dispatch
type eventManager struct {
	store Store
	lock  sync.RWMutex
}

// NewEventManager create a eventManager
func NewEventManager(store Store) Manager {
	manager := &eventManager{
		store: store,
	}

	store.SetManager(manager)

	return manager
}

// Listen create a relation from event to listners
func (em *eventManager) Listen(listeners ...Listener) {
	em.lock.Lock()
	defer em.lock.Unlock()

	for _, listener := range listeners {
		listenerType := reflect.TypeOf(listener)
		if listenerType.Kind() != reflect.Func {
			panic("listener must be a function")
		}

		if listenerType.NumIn() != 1 {
			panic("listener must be a function with only one arguemnt")
		}

		if listenerType.In(0).Kind() != reflect.Struct {
			panic("listener must be a function with only on argument of type struct")
		}

		em.store.Listen(fmt.Sprintf("%s", listenerType.In(0)), listener)
	}
}

// Publish a event
func (em *eventManager) Publish(evt interface{}) {
	em.lock.RLock()
	defer em.lock.RUnlock()

	em.store.Publish(fmt.Sprintf("%s", reflect.TypeOf(evt)), evt)
}

// Call trigger listener to execute
func (em *eventManager) Call(evt interface{}, listener Listener) {
	reflect.ValueOf(listener).Call([]reflect.Value{reflect.ValueOf(evt)})
}
