package backpressure

import (
	"sync"
	"sync/atomic"
	"time"
)

// Strategy defines backpressure strategy
type Strategy int

const (
	// StrategyDrop drops new messages when full
	StrategyDrop Strategy = iota
	// StrategyBlock blocks until space available
	StrategyBlock
	// StrategyDropOldest drops oldest messages
	StrategyDropOldest
)

// Controller manages backpressure for connections
type Controller struct {
	strategy Strategy

	// Metrics
	messagesDropped atomic.Int64
	messagesBlocked atomic.Int64

	// Thresholds
	highWaterMark int
	lowWaterMark  int

	// State
	pressured atomic.Bool

	// Callbacks
	onPressure   func()
	onRelief     func()
	callbackOnce sync.Once

	mu sync.RWMutex
}

// NewController creates a new backpressure controller
func NewController(strategy Strategy, highWaterMark, lowWaterMark int) *Controller {
	return &Controller{
		strategy:      strategy,
		highWaterMark: highWaterMark,
		lowWaterMark:  lowWaterMark,
	}
}

// CheckPressure checks if backpressure should be applied
func (c *Controller) CheckPressure(queueSize int) bool {
	if queueSize >= c.highWaterMark {
		if c.pressured.CompareAndSwap(false, true) {
			if c.onPressure != nil {
				go c.onPressure()
			}
		}
		return true
	}

	if queueSize <= c.lowWaterMark {
		if c.pressured.CompareAndSwap(true, false) {
			if c.onRelief != nil {
				go c.onRelief()
			}
		}
		return false
	}

	return c.pressured.Load()
}

// HandleBackpressure applies backpressure strategy
func (c *Controller) HandleBackpressure(queueSize int, timeout time.Duration) error {
	if !c.CheckPressure(queueSize) {
		return nil
	}

	switch c.strategy {
	case StrategyDrop:
		c.messagesDropped.Add(1)
		return ErrMessageDropped
	case StrategyBlock:
		c.messagesBlocked.Add(1)
		// Caller should handle blocking
		return ErrBackpressureApplied
	case StrategyDropOldest:
		// Caller should drop oldest message
		c.messagesDropped.Add(1)
		return ErrDropOldest
	}

	return nil
}

// IsPressured returns true if under pressure
func (c *Controller) IsPressured() bool {
	return c.pressured.Load()
}

// SetOnPressure sets the pressure callback
func (c *Controller) SetOnPressure(fn func()) {
	c.onPressure = fn
}

// SetOnRelief sets the relief callback
func (c *Controller) SetOnRelief(fn func()) {
	c.onRelief = fn
}

// Stats returns backpressure statistics
func (c *Controller) Stats() Stats {
	return Stats{
		Pressured:       c.pressured.Load(),
		MessagesDropped: c.messagesDropped.Load(),
		MessagesBlocked: c.messagesBlocked.Load(),
	}
}

// Stats represents backpressure statistics
type Stats struct {
	Pressured       bool
	MessagesDropped int64
	MessagesBlocked int64
}
