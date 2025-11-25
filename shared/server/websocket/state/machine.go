package state

import (
	"fmt"
	"sync"
)

// State represents connection state
type State int

const (
	// StateConnecting is initial state
	StateConnecting State = iota
	// StateConnected is active state
	StateConnected
	// StateDisconnecting is graceful shutdown state
	StateDisconnecting
	// StateDisconnected is final state
	StateDisconnected
	// StateReconnecting is attempting reconnect
	StateReconnecting
	// StateError is error state
	StateError
)

// String returns string representation of state
func (s State) String() string {
	switch s {
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateDisconnecting:
		return "disconnecting"
	case StateDisconnected:
		return "disconnected"
	case StateReconnecting:
		return "reconnecting"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Machine is a state machine for connection states
type Machine struct {
	current  State
	mu       sync.RWMutex
	onChange func(from, to State)
}

// NewMachine creates a new state machine
func NewMachine(initial State) *Machine {
	return &Machine{
		current: initial,
	}
}

// Current returns the current state
func (m *Machine) Current() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// Transition transitions to a new state
func (m *Machine) Transition(to State) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	from := m.current

	if !m.isValidTransition(from, to) {
		return fmt.Errorf("invalid state transition from %s to %s", from, to)
	}

	m.current = to

	if m.onChange != nil {
		go m.onChange(from, to)
	}

	return nil
}

// SetOnChange sets the state change callback
func (m *Machine) SetOnChange(fn func(from, to State)) {
	m.onChange = fn
}

// isValidTransition checks if transition is valid
func (m *Machine) isValidTransition(from, to State) bool {
	// Define valid transitions
	validTransitions := map[State][]State{
		StateConnecting: {StateConnected, StateDisconnecting, StateDisconnected, StateError},
		StateConnected:  {StateDisconnecting, StateDisconnected, StateError, StateReconnecting},
		StateDisconnecting: {StateDisconnected, StateError},
		StateDisconnected:  {StateReconnecting, StateConnecting},
		StateReconnecting:  {StateConnecting, StateDisconnected, StateError},
		StateError:         {StateDisconnected, StateReconnecting},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, state := range allowed {
		if state == to {
			return true
		}
	}

	return false
}
