package filter

import (
	"context"
)

// Message represents a filterable message
type Message struct {
	Type     string
	Data     []byte
	Metadata map[string]interface{}
}

// Filter is a message filter
type Filter interface {
	// Allow returns true if message should be processed
	Allow(ctx context.Context, msg *Message) bool
}

// Chain represents a filter chain
type Chain struct {
	filters []Filter
}

// NewChain creates a new filter chain
func NewChain(filters ...Filter) *Chain {
	return &Chain{
		filters: filters,
	}
}

// Add adds a filter to the chain
func (c *Chain) Add(filter Filter) {
	c.filters = append(c.filters, filter)
}

// Allow checks if all filters allow the message
func (c *Chain) Allow(ctx context.Context, msg *Message) bool {
	for _, filter := range c.filters {
		if !filter.Allow(ctx, msg) {
			return false
		}
	}
	return true
}
