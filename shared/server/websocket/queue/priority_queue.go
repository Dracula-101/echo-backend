package queue

import (
	"container/heap"
	"sync"
)

// PriorityQueue is a priority-based queue
type PriorityQueue struct {
	items   priorityHeap
	mu      sync.RWMutex
	maxSize int
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue(maxSize int) *PriorityQueue {
	pq := &PriorityQueue{
		items:   make(priorityHeap, 0),
		maxSize: maxSize,
	}
	heap.Init(&pq.items)
	return pq
}

// Enqueue adds a message with priority
func (pq *PriorityQueue) Enqueue(msg *Message) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.maxSize > 0 && len(pq.items) >= pq.maxSize {
		return ErrQueueFull
	}

	heap.Push(&pq.items, msg)
	return nil
}

// Dequeue removes and returns the highest priority message
func (pq *PriorityQueue) Dequeue() (*Message, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.items) == 0 {
		return nil, ErrQueueEmpty
	}

	return heap.Pop(&pq.items).(*Message), nil
}

// Size returns the queue size
func (pq *PriorityQueue) Size() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return len(pq.items)
}

// priorityHeap implements heap.Interface
type priorityHeap []*Message

func (h priorityHeap) Len() int           { return len(h) }
func (h priorityHeap) Less(i, j int) bool { return h[i].Priority > h[j].Priority }
func (h priorityHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *priorityHeap) Push(x interface{}) {
	*h = append(*h, x.(*Message))
}

func (h *priorityHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}
