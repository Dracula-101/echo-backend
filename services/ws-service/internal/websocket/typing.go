package websocket

import (
	"sync"
	"time"

	"shared/pkg/logger"

	"github.com/google/uuid"
)

// TypingIndicator represents a typing indicator
type TypingIndicator struct {
	UserID         uuid.UUID
	ConversationID uuid.UUID
	Timestamp      time.Time
}

// TypingManager manages typing indicators (application-specific)
type TypingManager struct {
	// conversation ID -> user ID -> timestamp
	indicators map[uuid.UUID]map[uuid.UUID]time.Time

	mu  sync.RWMutex
	log logger.Logger
}

// NewTypingManager creates a new typing manager
func NewTypingManager(log logger.Logger) *TypingManager {
	return &TypingManager{
		indicators: make(map[uuid.UUID]map[uuid.UUID]time.Time),
		log:        log,
	}
}

// StartTyping marks a user as typing in a conversation
func (tm *TypingManager) StartTyping(conversationID, userID uuid.UUID) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.indicators[conversationID] == nil {
		tm.indicators[conversationID] = make(map[uuid.UUID]time.Time)
	}

	tm.indicators[conversationID][userID] = time.Now()

	tm.log.Debug("User started typing",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", conversationID.String()),
	)
}

// StopTyping marks a user as stopped typing
func (tm *TypingManager) StopTyping(conversationID, userID uuid.UUID) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if users, ok := tm.indicators[conversationID]; ok {
		delete(users, userID)
		if len(users) == 0 {
			delete(tm.indicators, conversationID)
		}
	}

	tm.log.Debug("User stopped typing",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", conversationID.String()),
	)
}

// GetTypingUsers returns users currently typing in a conversation
func (tm *TypingManager) GetTypingUsers(conversationID uuid.UUID) []uuid.UUID {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	users, ok := tm.indicators[conversationID]
	if !ok {
		return nil
	}

	// Filter out stale indicators (older than 5 seconds)
	cutoff := time.Now().Add(-5 * time.Second)
	activeUsers := make([]uuid.UUID, 0)

	for userID, timestamp := range users {
		if timestamp.After(cutoff) {
			activeUsers = append(activeUsers, userID)
		}
	}

	return activeUsers
}

// Cleanup removes stale typing indicators
func (tm *TypingManager) Cleanup() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Second)

	for convID, users := range tm.indicators {
		for userID, timestamp := range users {
			if timestamp.Before(cutoff) {
				delete(users, userID)
			}
		}

		if len(users) == 0 {
			delete(tm.indicators, convID)
		}
	}
}
