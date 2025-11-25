package registry

import (
	"fmt"
	"sync"
)

// Item represents a registry item
type Item struct {
	Key   string
	Value interface{}
}

// Registry is a generic registry for storing items
type Registry struct {
	items map[string]interface{}
	mu    sync.RWMutex
}

// New creates a new registry
func New() *Registry {
	return &Registry{
		items: make(map[string]interface{}),
	}
}

// Register registers an item
func (r *Registry) Register(key string, value interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.items[key]; exists {
		return fmt.Errorf("item with key '%s' already exists", key)
	}

	r.items[key] = value
	return nil
}

// Unregister removes an item
func (r *Registry) Unregister(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, key)
}

// Get retrieves an item
func (r *Registry) Get(key string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, exists := r.items[key]
	return item, exists
}

// GetAll returns all items
func (r *Registry) GetAll() []Item {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Item, 0, len(r.items))
	for key, value := range r.items {
		items = append(items, Item{
			Key:   key,
			Value: value,
		})
	}
	return items
}

// Has checks if an item exists
func (r *Registry) Has(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.items[key]
	return exists
}

// Clear clears all items
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[string]interface{})
}

// Count returns the number of items
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.items)
}
