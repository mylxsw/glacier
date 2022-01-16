package event

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

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

// Listen create a relation from event to listeners
func (em *eventManager) Listen(listeners ...interface{}) {
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

// Publish an event
func (em *eventManager) Publish(evt interface{}) error {
	em.lock.RLock()
	defer em.lock.RUnlock()

	return em.store.Publish(Event{
		Name:  fmt.Sprintf("%s", reflect.TypeOf(evt)),
		Event: evt,
	})
}

// Call trigger listener to execute
func (em *eventManager) Call(evt interface{}, listener interface{}) {
	reflect.ValueOf(listener).Call([]reflect.Value{reflect.ValueOf(evt)})
}

func (em *eventManager) Start(ctx context.Context) <-chan interface{} {
	return em.store.Start(ctx)
}
