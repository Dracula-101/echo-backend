package filter

import (
	"context"
)

// TypeFilter filters messages by type
type TypeFilter struct {
	allowedTypes map[string]bool
}

// NewTypeFilter creates a new type filter
func NewTypeFilter(allowedTypes []string) *TypeFilter {
	allowed := make(map[string]bool)
	for _, t := range allowedTypes {
		allowed[t] = true
	}

	return &TypeFilter{
		allowedTypes: allowed,
	}
}

// Allow checks if message type is allowed
func (f *TypeFilter) Allow(ctx context.Context, msg *Message) bool {
	return f.allowedTypes[msg.Type]
}
