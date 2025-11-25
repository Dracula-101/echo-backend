package reconnect

import (
	"context"
	"math"
	"sync"
	"time"

	"shared/pkg/logger"
)

// Strategy defines reconnection strategy
type Strategy int

const (
	// StrategyExponential uses exponential backoff
	StrategyExponential Strategy = iota
	// StrategyLinear uses linear backoff
	StrategyLinear
	// StrategyConstant uses constant delay
	StrategyConstant
)

// Config holds reconnection configuration
type Config struct {
	Strategy        Strategy
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	Jitter          bool
}

// DefaultConfig returns default reconnection config
func DefaultConfig() *Config {
	return &Config{
		Strategy:     StrategyExponential,
		MaxAttempts:  5,
		InitialDelay: time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// Handler handles connection reconnection
type Handler struct {
	config     *Config
	attempts   map[string]int
	attemptsMu sync.RWMutex

	log logger.Logger
}

// NewHandler creates a new reconnection handler
func NewHandler(config *Config, log logger.Logger) *Handler {
	if config == nil {
		config = DefaultConfig()
	}

	return &Handler{
		config:   config,
		attempts: make(map[string]int),
		log:      log,
	}
}

// ShouldReconnect checks if reconnection should be attempted
func (h *Handler) ShouldReconnect(connID string) bool {
	h.attemptsMu.RLock()
	attempts := h.attempts[connID]
	h.attemptsMu.RUnlock()

	return h.config.MaxAttempts == 0 || attempts < h.config.MaxAttempts
}

// GetDelay calculates reconnection delay
func (h *Handler) GetDelay(connID string) time.Duration {
	h.attemptsMu.RLock()
	attempts := h.attempts[connID]
	h.attemptsMu.RUnlock()

	var delay time.Duration

	switch h.config.Strategy {
	case StrategyExponential:
		delay = h.exponentialDelay(attempts)
	case StrategyLinear:
		delay = h.linearDelay(attempts)
	case StrategyConstant:
		delay = h.config.InitialDelay
	}

	if h.config.Jitter {
		delay = h.applyJitter(delay)
	}

	if delay > h.config.MaxDelay {
		delay = h.config.MaxDelay
	}

	return delay
}

// IncrementAttempts increments reconnection attempts
func (h *Handler) IncrementAttempts(connID string) int {
	h.attemptsMu.Lock()
	defer h.attemptsMu.Unlock()

	h.attempts[connID]++
	return h.attempts[connID]
}

// ResetAttempts resets reconnection attempts
func (h *Handler) ResetAttempts(connID string) {
	h.attemptsMu.Lock()
	defer h.attemptsMu.Unlock()

	delete(h.attempts, connID)
}

// WaitForReconnect waits before reconnecting
func (h *Handler) WaitForReconnect(ctx context.Context, connID string) error {
	if !h.ShouldReconnect(connID) {
		return ErrMaxAttemptsReached
	}

	delay := h.GetDelay(connID)
	attempts := h.IncrementAttempts(connID)

	h.log.Info("Waiting for reconnection",
		logger.String("conn_id", connID),
		logger.Int("attempt", attempts),
		logger.Duration("delay", delay),
	)

	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// exponentialDelay calculates exponential backoff delay
func (h *Handler) exponentialDelay(attempts int) time.Duration {
	delay := float64(h.config.InitialDelay) * math.Pow(h.config.Multiplier, float64(attempts))
	return time.Duration(delay)
}

// linearDelay calculates linear backoff delay
func (h *Handler) linearDelay(attempts int) time.Duration {
	return h.config.InitialDelay * time.Duration(attempts+1)
}

// applyJitter adds jitter to delay
func (h *Handler) applyJitter(delay time.Duration) time.Duration {
	jitter := time.Duration(float64(delay) * 0.1) // 10% jitter
	return delay + jitter
}
