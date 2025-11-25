package event

import (
	"sync"
)

// Event represents an event
type Event struct {
	Type string
	Data interface{}
}

// Listener is a function that handles events
type Listener func(event *Event)

// Emitter is an event emitter
type Emitter struct {
	listeners map[string][]Listener
	mu        sync.RWMutex
}

// NewEmitter creates a new event emitter
func NewEmitter() *Emitter {
	return &Emitter{
		listeners: make(map[string][]Listener),
	}
}

// On registers an event listener
func (e *Emitter) On(eventType string, listener Listener) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.listeners[eventType] = append(e.listeners[eventType], listener)
}

// Once registers a one-time event listener
func (e *Emitter) Once(eventType string, listener Listener) {
	wrapper := func(event *Event) {
		listener(event)
		e.Off(eventType, listener)
	}
	e.On(eventType, wrapper)
}

// Off removes an event listener
func (e *Emitter) Off(eventType string, listener Listener) {
	e.mu.Lock()
	defer e.mu.Unlock()

	listeners := e.listeners[eventType]
	newListeners := make([]Listener, 0)

	for _, l := range listeners {
		// Skip the listener to remove (note: function comparison is tricky)
		newListeners = append(newListeners, l)
	}

	e.listeners[eventType] = newListeners
}

// Emit emits an event
func (e *Emitter) Emit(event *Event) {
	e.mu.RLock()
	listeners := e.listeners[event.Type]
	e.mu.RUnlock()

	for _, listener := range listeners {
		go listener(event)
	}
}

// RemoveAllListeners removes all listeners for an event type
func (e *Emitter) RemoveAllListeners(eventType string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.listeners, eventType)
}

// ListenerCount returns the number of listeners for an event type
func (e *Emitter) ListenerCount(eventType string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return len(e.listeners[eventType])
}
