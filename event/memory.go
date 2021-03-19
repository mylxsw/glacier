package event

import (
	"context"
)

// MemoryEventStore is a event store for sync operations
type MemoryEventStore struct {
	async       bool
	listeners   map[string][]interface{}
	manager     Manager
	asyncEvents chan Event
}

// NewMemoryEventStore create a sync event store
func NewMemoryEventStore(async bool, capacity int) Store {
	return &MemoryEventStore{
		async:       async,
		listeners:   make(map[string][]interface{}),
		asyncEvents: make(chan Event, capacity),
	}
}

// Listen add a listener to a event
func (eventStore *MemoryEventStore) Listen(evtType string, listener interface{}) {
	if _, ok := eventStore.listeners[evtType]; !ok {
		eventStore.listeners[evtType] = make([]interface{}, 0)
	}

	eventStore.listeners[evtType] = append(eventStore.listeners[evtType], listener)
}

// Publish publish a event
func (eventStore *MemoryEventStore) Publish(evt Event) {
	if eventStore.isAsyncEvent(evt.Event) {
		eventStore.asyncEvents <- evt
		return
	}

	eventStore.callEvent(evt)
}

func (eventStore *MemoryEventStore) callEvent(evt Event) {
	if listeners, ok := eventStore.listeners[evt.Name]; ok {
		for _, listener := range listeners {
			eventStore.manager.Call(evt.Event, listener)
		}
	}
}

// isAsyncEvent check whether the event is a async event
func (eventStore *MemoryEventStore) isAsyncEvent(evt interface{}) bool {
	if eventStore.async {
		return true
	}

	asyncEvent, ok := evt.(AsyncEvent)
	return ok && asyncEvent.Async()
}

// SetManager event manager
func (eventStore *MemoryEventStore) SetManager(manager Manager) {
	eventStore.manager = manager
}

func (eventStore *MemoryEventStore) Start(ctx context.Context) <-chan interface{} {
	stopped := make(chan interface{}, 0)

	go func() {
		for {
			select {
			case <-ctx.Done():
				for {
					select {
					case evt := <-eventStore.asyncEvents:
						eventStore.callEvent(evt)
					default:
						stopped <- struct{}{}
						return
					}
				}
			case evt := <-eventStore.asyncEvents:
				eventStore.callEvent(evt)
			}
		}
	}()

	return stopped
}
