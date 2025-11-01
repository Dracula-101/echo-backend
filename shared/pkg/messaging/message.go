package messaging

import (
	"time"
)

type Message struct {
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Topic     string
	Partition int32
	Offset    int64
	Timestamp time.Time
	Metadata  map[string]interface{}
}

func NewMessage(value []byte) *Message {
	return &Message{
		Value:     value,
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

func (m *Message) WithKey(key []byte) *Message {
	m.Key = key
	return m
}

func (m *Message) WithHeader(key, value string) *Message {
	m.Headers[key] = value
	return m
}

func (m *Message) WithHeaders(headers map[string]string) *Message {
	m.Headers = headers
	return m
}

func (m *Message) WithMetadata(key string, value interface{}) *Message {
	m.Metadata[key] = value
	return m
}
