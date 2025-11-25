package queue

import (
	"container/list"
	"sync"
	"time"
)

// Message represents a queued message
type Message struct {
	ID        string
	Data      []byte
	Priority  int
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// Queue is a thread-safe message queue
type Queue struct {
	items    *list.List
	mu       sync.RWMutex
	maxSize  int
	notEmpty *sync.Cond
}

// New creates a new queue
func New(maxSize int) *Queue {
	q := &Queue{
		items:   list.New(),
		maxSize: maxSize,
	}
	q.notEmpty = sync.NewCond(&q.mu)
	return q
}

// Enqueue adds a message to the queue
func (q *Queue) Enqueue(msg *Message) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.maxSize > 0 && q.items.Len() >= q.maxSize {
		return ErrQueueFull
	}

	q.items.PushBack(msg)
	q.notEmpty.Signal()
	return nil
}

// Dequeue removes and returns the first message
func (q *Queue) Dequeue() (*Message, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.items.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	elem := q.items.Front()
	q.items.Remove(elem)
	return elem.Value.(*Message), nil
}

// DequeueOrWait waits for a message if queue is empty
func (q *Queue) DequeueOrWait() *Message {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.items.Len() == 0 {
		q.notEmpty.Wait()
	}

	elem := q.items.Front()
	q.items.Remove(elem)
	return elem.Value.(*Message)
}

// Peek returns the first message without removing it
func (q *Queue) Peek() (*Message, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.items.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	return q.items.Front().Value.(*Message), nil
}

// Size returns the queue size
func (q *Queue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.items.Len()
}

// Clear clears the queue
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items.Init()
}

// IsEmpty returns true if queue is empty
func (q *Queue) IsEmpty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.items.Len() == 0
}

// IsFull returns true if queue is full
func (q *Queue) IsFull() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.maxSize > 0 && q.items.Len() >= q.maxSize
}
