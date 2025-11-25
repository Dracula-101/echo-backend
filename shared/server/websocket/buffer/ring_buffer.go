package buffer

import (
	"sync"
)

// RingBuffer is a circular buffer for messages
type RingBuffer struct {
	data  [][]byte
	head  int
	tail  int
	size  int
	count int
	mu    sync.RWMutex
}

// NewRingBuffer creates a new ring buffer
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([][]byte, size),
		size: size,
	}
}

// Write writes data to the buffer
func (rb *RingBuffer) Write(data []byte) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count >= rb.size {
		return ErrBufferFull
	}

	// Copy data to avoid external modifications
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	rb.data[rb.tail] = dataCopy
	rb.tail = (rb.tail + 1) % rb.size
	rb.count++

	return nil
}

// Read reads data from the buffer
func (rb *RingBuffer) Read() ([]byte, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return nil, ErrBufferEmpty
	}

	data := rb.data[rb.head]
	rb.data[rb.head] = nil // Clear reference
	rb.head = (rb.head + 1) % rb.size
	rb.count--

	return data, nil
}

// Peek returns the next item without removing it
func (rb *RingBuffer) Peek() ([]byte, error) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 {
		return nil, ErrBufferEmpty
	}

	return rb.data[rb.head], nil
}

// Len returns the number of items in the buffer
func (rb *RingBuffer) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Cap returns the capacity of the buffer
func (rb *RingBuffer) Cap() int {
	return rb.size
}

// Clear clears the buffer
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.head = 0
	rb.tail = 0
	rb.count = 0
	rb.data = make([][]byte, rb.size)
}

// IsFull returns true if buffer is full
func (rb *RingBuffer) IsFull() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count >= rb.size
}

// IsEmpty returns true if buffer is empty
func (rb *RingBuffer) IsEmpty() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count == 0
}
