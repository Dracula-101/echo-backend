package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"shared/pkg/logger"
	"sync"
	"syscall"
	"time"
)

// Hook represents a shutdown hook function
type Hook func(context.Context) error

// Priority defines the execution order of hooks
type Priority int

const (
	// PriorityHigh hooks run first (e.g., stop accepting new requests)
	PriorityHigh Priority = 100

	// PriorityNormal hooks run second (e.g., finish processing existing requests)
	PriorityNormal Priority = 50

	// PriorityLow hooks run last (e.g., cleanup resources, close connections)
	PriorityLow Priority = 10
)

// HookEntry represents a registered shutdown hook with metadata
type HookEntry struct {
	Name     string
	Priority Priority
	Hook     Hook
	Timeout  time.Duration
}

// Manager manages graceful shutdown
type Manager struct {
	hooks    []HookEntry
	mu       sync.RWMutex
	stopping bool
	signals  []os.Signal
	timeout  time.Duration
	logger   Logger
}

// Logger interface for logging shutdown events
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, err error, keysAndValues ...interface{})
}

// noopLogger is a no-op logger implementation
type noopLogger struct{}

func (n *noopLogger) Info(msg string, keysAndValues ...interface{})             {}
func (n *noopLogger) Error(msg string, err error, keysAndValues ...interface{}) {}

// Option is a functional option for configuring Manager
type Option func(*Manager)

// WithSignals sets the signals that trigger shutdown
func WithSignals(signals ...os.Signal) Option {
	return func(m *Manager) {
		m.signals = signals
	}
}

// WithTimeout sets the global timeout for shutdown
func WithTimeout(timeout time.Duration) Option {
	return func(m *Manager) {
		m.timeout = timeout
	}
}

// WithLogger sets the logger for shutdown events
func WithLogger(logger logger.Logger) Option {
	return func(m *Manager) {
		m.logger = &shutdownLogger{log: logger}
	}
}

// New creates a new shutdown manager
func New(opts ...Option) *Manager {
	m := &Manager{
		hooks:   make([]HookEntry, 0),
		signals: []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT},
		timeout: 30 * time.Second,
		logger:  &noopLogger{},
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Register adds a shutdown hook with default priority and timeout
func (m *Manager) Register(name string, hook Hook) {
	m.RegisterWithOptions(name, hook, PriorityNormal, 0)
}

// RegisterWithPriority adds a shutdown hook with specified priority
func (m *Manager) RegisterWithPriority(name string, hook Hook, priority Priority) {
	m.RegisterWithOptions(name, hook, priority, 0)
}

// RegisterWithOptions adds a shutdown hook with full options
func (m *Manager) RegisterWithOptions(name string, hook Hook, priority Priority, timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopping {
		m.logger.Info("cannot register hook during shutdown", "name", name)
		return
	}

	entry := HookEntry{
		Name:     name,
		Priority: priority,
		Hook:     hook,
		Timeout:  timeout,
	}

	// Insert in priority order (highest priority first)
	inserted := false
	for i, existing := range m.hooks {
		if priority > existing.Priority {
			m.hooks = append(m.hooks[:i], append([]HookEntry{entry}, m.hooks[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		m.hooks = append(m.hooks, entry)
	}

	m.logger.Info("registered shutdown hook", "name", name, "priority", priority)
}

// Wait blocks until a shutdown signal is received, then executes all hooks
func (m *Manager) Wait() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, m.signals...)

	sig := <-sigChan
	m.logger.Info("received shutdown signal", "signal", sig.String())

	return m.Shutdown(context.Background())
}

// Shutdown executes all registered hooks in priority order
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	if m.stopping {
		m.mu.Unlock()
		return fmt.Errorf("shutdown already in progress")
	}
	m.stopping = true
	hooks := make([]HookEntry, len(m.hooks))
	copy(hooks, m.hooks)
	m.mu.Unlock()

	m.logger.Info("starting graceful shutdown", "hooks", len(hooks), "timeout", m.timeout)

	// Create a context with overall timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	errors := make([]error, 0)
	var errorsMu sync.Mutex

	// Execute hooks in priority order
	for _, entry := range hooks {
		// Use hook-specific timeout if set, otherwise use remaining time
		hookCtx := shutdownCtx
		if entry.Timeout > 0 {
			var hookCancel context.CancelFunc
			hookCtx, hookCancel = context.WithTimeout(shutdownCtx, entry.Timeout)
			defer hookCancel()
		}

		m.logger.Info("executing shutdown hook", "name", entry.Name, "priority", entry.Priority)
		startTime := time.Now()

		err := m.executeHook(hookCtx, entry)
		duration := time.Since(startTime)

		if err != nil {
			m.logger.Error("shutdown hook failed", err, "name", entry.Name, "duration", duration)
			errorsMu.Lock()
			errors = append(errors, fmt.Errorf("%s: %w", entry.Name, err))
			errorsMu.Unlock()
		} else {
			m.logger.Info("shutdown hook completed", "name", entry.Name, "duration", duration)
		}

		// Check if overall shutdown context is done
		select {
		case <-shutdownCtx.Done():
			m.logger.Error("shutdown timeout exceeded", shutdownCtx.Err())
			return fmt.Errorf("shutdown timeout exceeded: %w", shutdownCtx.Err())
		default:
		}
	}

	if len(errors) > 0 {
		m.logger.Error("shutdown completed with errors", fmt.Errorf("%d hooks failed", len(errors)))
		return &ShutdownErrors{Errors: errors}
	}

	m.logger.Info("graceful shutdown completed successfully")
	return nil
}

func (m *Manager) executeHook(ctx context.Context, entry HookEntry) error {
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in shutdown hook: %v", r)
			}
		}()
		errChan <- entry.Hook(ctx)
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("hook timeout: %w", ctx.Err())
	}
}

// IsShuttingDown returns true if shutdown has been initiated
func (m *Manager) IsShuttingDown() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopping
}

// ShutdownErrors represents multiple errors during shutdown
type ShutdownErrors struct {
	Errors []error
}

func (e *ShutdownErrors) Error() string {
	return fmt.Sprintf("shutdown completed with %d errors: %v", len(e.Errors), e.Errors)
}

// Unwrap returns the underlying errors
func (e *ShutdownErrors) Unwrap() []error {
	return e.Errors
}

// Common shutdown hooks

// ServerShutdownHook creates a hook for shutting down an HTTP server
func ServerShutdownHook(server interface{ Shutdown(context.Context) error }) Hook {
	return func(ctx context.Context) error {
		return server.Shutdown(ctx)
	}
}

// ConnectionPoolShutdownHook creates a hook for closing a connection pool
func ConnectionPoolShutdownHook(pool interface{ Close() error }) Hook {
	return func(ctx context.Context) error {
		done := make(chan error, 1)
		go func() {
			done <- pool.Close()
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// WorkerShutdownHook creates a hook for stopping background workers
func WorkerShutdownHook(stopFunc func() error) Hook {
	return func(ctx context.Context) error {
		done := make(chan error, 1)
		go func() {
			done <- stopFunc()
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ResourceCleanupHook creates a hook for cleaning up resources
func ResourceCleanupHook(cleanupFunc func() error) Hook {
	return func(ctx context.Context) error {
		done := make(chan error, 1)
		go func() {
			done <- cleanupFunc()
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// DelayHook creates a hook that waits for a specified duration
// Useful for ensuring in-flight requests complete
func DelayHook(duration time.Duration) Hook {
	return func(ctx context.Context) error {
		select {
		case <-time.After(duration):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ParallelHook combines multiple hooks to run in parallel
func ParallelHook(hooks ...Hook) Hook {
	return func(ctx context.Context) error {
		var wg sync.WaitGroup
		errors := make([]error, 0)
		var mu sync.Mutex

		for i, hook := range hooks {
			wg.Add(1)
			go func(index int, h Hook) {
				defer wg.Done()
				if err := h(ctx); err != nil {
					mu.Lock()
					errors = append(errors, fmt.Errorf("parallel hook %d: %w", index, err))
					mu.Unlock()
				}
			}(i, hook)
		}

		wg.Wait()

		if len(errors) > 0 {
			return &ShutdownErrors{Errors: errors}
		}

		return nil
	}
}

// SequentialHook combines multiple hooks to run in sequence
func SequentialHook(hooks ...Hook) Hook {
	return func(ctx context.Context) error {
		for i, hook := range hooks {
			if err := hook(ctx); err != nil {
				return fmt.Errorf("sequential hook %d: %w", i, err)
			}

			// Check context between hooks
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		return nil
	}
}

// ConditionalHook runs a hook only if the condition is true
func ConditionalHook(condition func() bool, hook Hook) Hook {
	return func(ctx context.Context) error {
		if condition() {
			return hook(ctx)
		}
		return nil
	}
}

// Global default manager
var defaultManager = New()

// Register adds a hook to the default manager
func Register(name string, hook Hook) {
	defaultManager.Register(name, hook)
}

// RegisterWithPriority adds a hook with priority to the default manager
func RegisterWithPriority(name string, hook Hook, priority Priority) {
	defaultManager.RegisterWithPriority(name, hook, priority)
}

// Wait blocks until shutdown signal on the default manager
func Wait() error {
	return defaultManager.Wait()
}

// Shutdown executes shutdown on the default manager
func Shutdown(ctx context.Context) error {
	return defaultManager.Shutdown(ctx)
}

// IsShuttingDown checks if the default manager is shutting down
func IsShuttingDown() bool {
	return defaultManager.IsShuttingDown()
}
