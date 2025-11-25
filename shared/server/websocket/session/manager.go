package session

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents a WebSocket session
type Session struct {
	ID        string
	UserID    uuid.UUID
	CreatedAt time.Time
	ExpiresAt time.Time
	Data      map[string]interface{}
	mu        sync.RWMutex
}

// Get retrieves session data
func (s *Session) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.Data[key]
	return val, ok
}

// Set sets session data
func (s *Session) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[key] = value
}

// Delete removes session data
func (s *Session) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Data, key)
}

// IsExpired checks if session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Manager manages WebSocket sessions
type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex

	defaultTTL time.Duration

	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// NewManager creates a new session manager
func NewManager(defaultTTL, cleanupInterval time.Duration) *Manager {
	return &Manager{
		sessions:        make(map[string]*Session),
		defaultTTL:      defaultTTL,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}
}

// Create creates a new session
func (m *Manager) Create(userID uuid.UUID, ttl time.Duration) *Session {
	if ttl == 0 {
		ttl = m.defaultTTL
	}

	session := &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		Data:      make(map[string]interface{}),
	}

	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	return session
}

// Get retrieves a session by ID
func (m *Manager) Get(sessionID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists || session.IsExpired() {
		return nil, false
	}

	return session, true
}

// Delete deletes a session
func (m *Manager) Delete(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

// Renew renews a session's expiration
func (m *Manager) Renew(sessionID string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	if ttl == 0 {
		ttl = m.defaultTTL
	}

	session.ExpiresAt = time.Now().Add(ttl)
	return nil
}

// StartCleanup starts periodic cleanup of expired sessions
func (m *Manager) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCleanup:
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

// StopCleanup stops the cleanup goroutine
func (m *Manager) StopCleanup() {
	close(m.stopCleanup)
}

// cleanup removes expired sessions
func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	expired := make([]string, 0)
	for id, session := range m.sessions {
		if session.IsExpired() {
			expired = append(expired, id)
		}
	}

	for _, id := range expired {
		delete(m.sessions, id)
	}
}

// Count returns the number of active sessions
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
