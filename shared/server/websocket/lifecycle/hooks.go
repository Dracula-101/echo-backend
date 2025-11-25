package lifecycle

import (
	"context"
	"sync"
)

// Hook is a lifecycle hook function
type Hook func(ctx context.Context, data interface{}) error

// Phase represents lifecycle phase
type Phase string

const (
	// PhaseBeforeConnect is before connection establishment
	PhaseBeforeConnect Phase = "before_connect"
	// PhaseAfterConnect is after connection establishment
	PhaseAfterConnect Phase = "after_connect"
	// PhaseBeforeDisconnect is before disconnection
	PhaseBeforeDisconnect Phase = "before_disconnect"
	// PhaseAfterDisconnect is after disconnection
	PhaseAfterDisconnect Phase = "after_disconnect"
	// PhaseBeforeMessage is before message processing
	PhaseBeforeMessage Phase = "before_message"
	// PhaseAfterMessage is after message processing
	PhaseAfterMessage Phase = "after_message"
	// PhaseOnError is when error occurs
	PhaseOnError Phase = "on_error"
)

// Manager manages lifecycle hooks
type Manager struct {
	hooks map[Phase][]Hook
	mu    sync.RWMutex
}

// NewManager creates a new lifecycle manager
func NewManager() *Manager {
	return &Manager{
		hooks: make(map[Phase][]Hook),
	}
}

// Register registers a hook for a phase
func (m *Manager) Register(phase Phase, hook Hook) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hooks[phase] = append(m.hooks[phase], hook)
}

// Execute executes all hooks for a phase
func (m *Manager) Execute(ctx context.Context, phase Phase, data interface{}) error {
	m.mu.RLock()
	hooks := m.hooks[phase]
	m.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook(ctx, data); err != nil {
			return err
		}
	}

	return nil
}

// Clear clears all hooks for a phase
func (m *Manager) Clear(phase Phase) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.hooks, phase)
}
