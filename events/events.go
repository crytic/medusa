package events

import (
	"reflect"
	"sync"
)

// EventHandler defines a function type where its input type is the generic type.
type EventHandler[T any] func(T) error

// EventType returns the event type given an EventHandler object
func (e *EventHandler[T]) EventType() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// globalEventHandlers describes a mapping of event types to EventHandler objects. These callbacks are called
// any time any EventEmitter publishes an event of that type.
var globalEventHandlers map[string][]any

// globalEventHandlersLock is a lock that provides thread synchronization when accessing globalEventHandlers. This
// helps in avoiding concurrent access panics.
var globalEventHandlersLock sync.Mutex

// SubscribeAny adds an EventHandler to the list of global EventHandler objects for this a given event data type.
// When an event is published, the callback will be triggered with the event data.
// Note: An EventHandler subscribed here will remain throughout program execution. Objects which should be freed from
// memory should not use this method to avoid memory leaks.
func SubscribeAny[T any](callback EventHandler[T]) {
	// Obtain the type of event this handler handles.
	eventType := callback.EventType()

	// If our global event handlers are nil, instantiate them.
	if globalEventHandlers == nil {
		globalEventHandlers = make(map[string][]any, 0)
	}

	// Acquire a thread lock for the next few operations to avoid concurrent access panics.
	globalEventHandlersLock.Lock()
	defer globalEventHandlersLock.Unlock()

	// If we don't have an event handlers list for an event of this type, create it.
	if _, ok := globalEventHandlers[eventType.String()]; !ok {
		globalEventHandlers[eventType.String()] = make([]any, 0)
	}

	// Add our callback to the event handlers list for events of this type.
	globalEventHandlers[eventType.String()] = append(globalEventHandlers[eventType.String()], callback)
}

// EventEmitter describes a provider which can subscribe EventHandler methods for callback when the event type (generic)
// is published. It additionally provides methods for publishing events.
type EventEmitter[T any] struct {
	// subscriptions defines the EventHandler methods which should be invoked when a new event is published to this
	// emitter.
	subscriptions []EventHandler[T]
}

// EventType returns the event type given an EventEmitter object
func (e *EventEmitter[T]) EventType() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// Publish emits the provided event by calling every EventHandler subscribed.
func (e *EventEmitter[T]) Publish(event T) error {
	// Call every subscribed EventHandler
	for _, subscription := range e.subscriptions {
		err := subscription(event)
		if err != nil {
			return err
		}
	}

	// Determine the event type
	eventType := reflect.TypeOf(event)

	// If we have any handlers, invoke them.
	if globalEventHandlers != nil {
		// Acquire a thread lock when fetching our event handlers to avoid concurrent access panics.
		globalEventHandlersLock.Lock()
		callbacks := globalEventHandlers[eventType.String()]
		globalEventHandlersLock.Unlock()

		// Call all relevant event handlers.
		for i := 0; i < len(callbacks); i++ {
			callback := callbacks[i].(EventHandler[T])
			err := callback(event)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Subscribe adds an EventHandler to the list of subscribed EventHandler objects for this emitter. When an event is
// published, the callback will be triggered with the event data.
func (e *EventEmitter[T]) Subscribe(callback EventHandler[T]) {
	e.subscriptions = append(e.subscriptions, callback)
}
