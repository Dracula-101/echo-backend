package redis

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"shared/pkg/logger"
)

// KeyspaceListener subscribes to Redis keyspace expiry events on a single DB.
// It's a safety net for cleanup that didn't fire via the in-memory paths —
// e.g. when a pod crashes mid-flight and a typing key expires with no one
// watching. Each registered handler is invoked with the FULL expired key.
type KeyspaceListener struct {
	client *Client
	db     int
	log    logger.Logger

	mu       sync.RWMutex
	handlers map[string]ExpiryHandler // prefix → handler

	stop chan struct{}
	wg   sync.WaitGroup
}

// ExpiryHandler is invoked with the expired key (e.g. "ws:typing:abc-123").
// Handlers must be fast and non-blocking — the listener has a single goroutine
// pumping the subscribe channel.
type ExpiryHandler func(key string)

func NewKeyspaceListener(client *Client, db int, log logger.Logger) *KeyspaceListener {
	return &KeyspaceListener{
		client:   client,
		db:       db,
		log:      log,
		handlers: make(map[string]ExpiryHandler),
		stop:     make(chan struct{}),
	}
}

// OnPrefixExpired registers a handler invoked when any key starting with
// prefix expires.
func (l *KeyspaceListener) OnPrefixExpired(prefix string, h ExpiryHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers[prefix] = h
}

func (l *KeyspaceListener) Start(ctx context.Context) error {
	channel := fmt.Sprintf("__keyevent@%d__:expired", l.db)
	pubsub := l.client.Subscribe(ctx, channel)

	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		ch := pubsub.Channel()
		for {
			select {
			case <-l.stop:
				_ = pubsub.Close()
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				l.dispatch(msg.Payload)
			}
		}
	}()

	l.log.Info("keyspace listener started",
		logger.String("channel", channel),
	)
	return nil
}

func (l *KeyspaceListener) Stop() {
	close(l.stop)
	l.wg.Wait()
}

func (l *KeyspaceListener) dispatch(key string) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for prefix, h := range l.handlers {
		if strings.HasPrefix(key, prefix) {
			h(key)
		}
	}
}
