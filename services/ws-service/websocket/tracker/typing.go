package tracker

import (
	"sync"
	"time"

	"shared/pkg/logger"

	"github.com/google/uuid"
)

type TypingIndicator struct {
	UserID         uuid.UUID `json:"user_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	StartedAt      time.Time `json:"started_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type TypingEvent struct {
	UserID         uuid.UUID `json:"user_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	IsTyping       bool      `json:"is_typing"`
	Timestamp      time.Time `json:"timestamp"`
}

type TypingEventCallback func(event TypingEvent)

type TypingManager struct {
	indicators map[uuid.UUID]map[uuid.UUID]*TypingIndicator
	callbacks  []TypingEventCallback
	timeout    time.Duration
	mu         sync.RWMutex
	log        logger.Logger
}

type TypingManagerConfig struct {
	Timeout time.Duration
}

func NewTypingManager(log logger.Logger, cfg *TypingManagerConfig) *TypingManager {
	timeout := 5 * time.Second
	if cfg != nil && cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	return &TypingManager{
		indicators: make(map[uuid.UUID]map[uuid.UUID]*TypingIndicator),
		callbacks:  make([]TypingEventCallback, 0),
		timeout:    timeout,
		log:        log,
	}
}

func (tm *TypingManager) OnTypingEvent(callback TypingEventCallback) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.callbacks = append(tm.callbacks, callback)
}

func (tm *TypingManager) notifyEvent(event TypingEvent) {
	for _, cb := range tm.callbacks {
		go cb(event)
	}
}

func (tm *TypingManager) StartTyping(conversationID, userID uuid.UUID) TypingEvent {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()

	if tm.indicators[conversationID] == nil {
		tm.indicators[conversationID] = make(map[uuid.UUID]*TypingIndicator)
	}

	existing := tm.indicators[conversationID][userID]
	isNew := existing == nil

	tm.indicators[conversationID][userID] = &TypingIndicator{
		UserID:         userID,
		ConversationID: conversationID,
		StartedAt:      now,
		ExpiresAt:      now.Add(tm.timeout),
	}

	tm.log.Debug("User started typing",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", conversationID.String()),
	)

	event := TypingEvent{
		UserID:         userID,
		ConversationID: conversationID,
		IsTyping:       true,
		Timestamp:      now,
	}

	if isNew {
		tm.notifyEvent(event)
	}

	return event
}

func (tm *TypingManager) StopTyping(conversationID, userID uuid.UUID) TypingEvent {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()

	if users, ok := tm.indicators[conversationID]; ok {
		if _, exists := users[userID]; exists {
			delete(users, userID)
			if len(users) == 0 {
				delete(tm.indicators, conversationID)
			}

			tm.log.Debug("User stopped typing",
				logger.String("user_id", userID.String()),
				logger.String("conversation_id", conversationID.String()),
			)

			event := TypingEvent{
				UserID:         userID,
				ConversationID: conversationID,
				IsTyping:       false,
				Timestamp:      now,
			}

			tm.notifyEvent(event)
			return event
		}
	}

	return TypingEvent{}
}

func (tm *TypingManager) StopTypingForUser(userID uuid.UUID) []TypingEvent {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	events := make([]TypingEvent, 0)

	for convID, users := range tm.indicators {
		if _, exists := users[userID]; exists {
			delete(users, userID)
			if len(users) == 0 {
				delete(tm.indicators, convID)
			}

			event := TypingEvent{
				UserID:         userID,
				ConversationID: convID,
				IsTyping:       false,
				Timestamp:      now,
			}
			events = append(events, event)
		}
	}

	for _, event := range events {
		tm.notifyEvent(event)
	}

	return events
}

func (tm *TypingManager) GetTypingUsers(conversationID uuid.UUID) []TypingIndicator {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	users, ok := tm.indicators[conversationID]
	if !ok {
		return nil
	}

	now := time.Now()
	activeUsers := make([]TypingIndicator, 0, len(users))

	for _, indicator := range users {
		if indicator.ExpiresAt.After(now) {
			activeUsers = append(activeUsers, *indicator)
		}
	}

	return activeUsers
}

func (tm *TypingManager) IsUserTyping(conversationID, userID uuid.UUID) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if users, ok := tm.indicators[conversationID]; ok {
		if indicator, exists := users[userID]; exists {
			return indicator.ExpiresAt.After(time.Now())
		}
	}
	return false
}

func (tm *TypingManager) Cleanup() []TypingEvent {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	events := make([]TypingEvent, 0)

	for convID, users := range tm.indicators {
		for userID, indicator := range users {
			if indicator.ExpiresAt.Before(now) {
				delete(users, userID)

				event := TypingEvent{
					UserID:         userID,
					ConversationID: convID,
					IsTyping:       false,
					Timestamp:      now,
				}
				events = append(events, event)
			}
		}

		if len(users) == 0 {
			delete(tm.indicators, convID)
		}
	}

	for _, event := range events {
		tm.notifyEvent(event)
	}

	return events
}

func (tm *TypingManager) GetStats() TypingStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	now := time.Now()
	activeIndicators := 0

	for _, users := range tm.indicators {
		for _, indicator := range users {
			if indicator.ExpiresAt.After(now) {
				activeIndicators++
			}
		}
	}

	return TypingStats{
		ActiveConversations: len(tm.indicators),
		ActiveIndicators:    activeIndicators,
	}
}

type TypingStats struct {
	ActiveConversations int `json:"active_conversations"`
	ActiveIndicators    int `json:"active_indicators"`
}
