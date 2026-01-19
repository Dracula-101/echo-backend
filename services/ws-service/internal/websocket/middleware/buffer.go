package middleware

import (
	"container/list"
	"sync"
	"time"
)

type BufferedMessage struct {
	ID           string
	Data         []byte
	CreatedAt    time.Time
	Attempts     int
	LastAttempt  time.Time
	Acknowledged bool
}

type MessageBufferConfig struct {
	MaxSize       int
	MaxAge        time.Duration
	MaxRetries    int
	RetryDelay    time.Duration
	CleanupInterval time.Duration
}

func DefaultMessageBufferConfig() *MessageBufferConfig {
	return &MessageBufferConfig{
		MaxSize:       100,
		MaxAge:        5 * time.Minute,
		MaxRetries:    3,
		RetryDelay:    100 * time.Millisecond,
		CleanupInterval: 30 * time.Second,
	}
}

type MessageBuffer struct {
	buffers map[string]*connectionBuffer
	config  *MessageBufferConfig
	mu      sync.RWMutex
	stopCh  chan struct{}
}

type connectionBuffer struct {
	messages *list.List
	index    map[string]*list.Element
	mu       sync.Mutex
}

func NewMessageBuffer(cfg *MessageBufferConfig) *MessageBuffer {
	if cfg == nil {
		cfg = DefaultMessageBufferConfig()
	}

	mb := &MessageBuffer{
		buffers: make(map[string]*connectionBuffer),
		config:  cfg,
		stopCh:  make(chan struct{}),
	}

	go mb.runCleanup()
	return mb
}

func (mb *MessageBuffer) getBuffer(connID string) *connectionBuffer {
	mb.mu.RLock()
	buf, exists := mb.buffers[connID]
	mb.mu.RUnlock()

	if exists {
		return buf
	}

	mb.mu.Lock()
	defer mb.mu.Unlock()

	if buf, exists = mb.buffers[connID]; exists {
		return buf
	}

	buf = &connectionBuffer{
		messages: list.New(),
		index:    make(map[string]*list.Element),
	}
	mb.buffers[connID] = buf
	return buf
}

func (mb *MessageBuffer) Add(connID, messageID string, data []byte) bool {
	buf := mb.getBuffer(connID)

	buf.mu.Lock()
	defer buf.mu.Unlock()

	if _, exists := buf.index[messageID]; exists {
		return false
	}

	if buf.messages.Len() >= mb.config.MaxSize {
		oldest := buf.messages.Front()
		if oldest != nil {
			oldMsg := oldest.Value.(*BufferedMessage)
			delete(buf.index, oldMsg.ID)
			buf.messages.Remove(oldest)
		}
	}

	msg := &BufferedMessage{
		ID:        messageID,
		Data:      data,
		CreatedAt: time.Now(),
		Attempts:  0,
	}

	elem := buf.messages.PushBack(msg)
	buf.index[messageID] = elem
	return true
}

func (mb *MessageBuffer) Acknowledge(connID, messageID string) bool {
	mb.mu.RLock()
	buf, exists := mb.buffers[connID]
	mb.mu.RUnlock()

	if !exists {
		return false
	}

	buf.mu.Lock()
	defer buf.mu.Unlock()

	if elem, exists := buf.index[messageID]; exists {
		msg := elem.Value.(*BufferedMessage)
		msg.Acknowledged = true
		delete(buf.index, messageID)
		buf.messages.Remove(elem)
		return true
	}

	return false
}

func (mb *MessageBuffer) GetPending(connID string) []*BufferedMessage {
	mb.mu.RLock()
	buf, exists := mb.buffers[connID]
	mb.mu.RUnlock()

	if !exists {
		return nil
	}

	buf.mu.Lock()
	defer buf.mu.Unlock()

	now := time.Now()
	pending := make([]*BufferedMessage, 0)

	for elem := buf.messages.Front(); elem != nil; elem = elem.Next() {
		msg := elem.Value.(*BufferedMessage)

		if msg.Acknowledged {
			continue
		}

		if now.Sub(msg.CreatedAt) > mb.config.MaxAge {
			continue
		}

		if msg.Attempts >= mb.config.MaxRetries {
			continue
		}

		if msg.Attempts > 0 && now.Sub(msg.LastAttempt) < mb.config.RetryDelay {
			continue
		}

		pending = append(pending, msg)
	}

	return pending
}

func (mb *MessageBuffer) MarkAttempt(connID, messageID string) {
	mb.mu.RLock()
	buf, exists := mb.buffers[connID]
	mb.mu.RUnlock()

	if !exists {
		return
	}

	buf.mu.Lock()
	defer buf.mu.Unlock()

	if elem, exists := buf.index[messageID]; exists {
		msg := elem.Value.(*BufferedMessage)
		msg.Attempts++
		msg.LastAttempt = time.Now()
	}
}

func (mb *MessageBuffer) Remove(connID, messageID string) {
	mb.mu.RLock()
	buf, exists := mb.buffers[connID]
	mb.mu.RUnlock()

	if !exists {
		return
	}

	buf.mu.Lock()
	defer buf.mu.Unlock()

	if elem, exists := buf.index[messageID]; exists {
		delete(buf.index, messageID)
		buf.messages.Remove(elem)
	}
}

func (mb *MessageBuffer) ClearConnection(connID string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	delete(mb.buffers, connID)
}

func (mb *MessageBuffer) runCleanup() {
	ticker := time.NewTicker(mb.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mb.cleanup()
		case <-mb.stopCh:
			return
		}
	}
}

func (mb *MessageBuffer) cleanup() {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	now := time.Now()

	for connID, buf := range mb.buffers {
		buf.mu.Lock()

		var next *list.Element
		for elem := buf.messages.Front(); elem != nil; elem = next {
			next = elem.Next()
			msg := elem.Value.(*BufferedMessage)

			if msg.Acknowledged || now.Sub(msg.CreatedAt) > mb.config.MaxAge || msg.Attempts >= mb.config.MaxRetries {
				delete(buf.index, msg.ID)
				buf.messages.Remove(elem)
			}
		}

		if buf.messages.Len() == 0 {
			delete(mb.buffers, connID)
		}

		buf.mu.Unlock()
	}
}

func (mb *MessageBuffer) Stop() {
	close(mb.stopCh)
}

func (mb *MessageBuffer) GetStats() MessageBufferStats {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	totalMessages := 0
	pendingMessages := 0

	for _, buf := range mb.buffers {
		buf.mu.Lock()
		totalMessages += buf.messages.Len()
		for elem := buf.messages.Front(); elem != nil; elem = elem.Next() {
			msg := elem.Value.(*BufferedMessage)
			if !msg.Acknowledged {
				pendingMessages++
			}
		}
		buf.mu.Unlock()
	}

	return MessageBufferStats{
		ActiveConnections: len(mb.buffers),
		TotalMessages:     totalMessages,
		PendingMessages:   pendingMessages,
	}
}

type MessageBufferStats struct {
	ActiveConnections int `json:"active_connections"`
	TotalMessages     int `json:"total_messages"`
	PendingMessages   int `json:"pending_messages"`
}
