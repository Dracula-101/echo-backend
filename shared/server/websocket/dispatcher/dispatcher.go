package dispatcher

import (
	"context"
	"sync"

	"shared/pkg/logger"
)

// Message represents a message to dispatch
type Message struct {
	Target  string
	Payload []byte
}

// Handler handles dispatched messages
type Handler func(ctx context.Context, msg *Message) error

// Dispatcher dispatches messages to workers
type Dispatcher struct {
	workers    int
	queue      chan *Message
	handlers   map[string]Handler
	handlersMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	log logger.Logger
}

// New creates a new dispatcher
func New(workers, queueSize int, log logger.Logger) *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())

	return &Dispatcher{
		workers:  workers,
		queue:    make(chan *Message, queueSize),
		handlers: make(map[string]Handler),
		ctx:      ctx,
		cancel:   cancel,
		log:      log,
	}
}

// RegisterHandler registers a handler for a target
func (d *Dispatcher) RegisterHandler(target string, handler Handler) {
	d.handlersMu.Lock()
	defer d.handlersMu.Unlock()
	d.handlers[target] = handler
}

// Dispatch dispatches a message
func (d *Dispatcher) Dispatch(msg *Message) error {
	select {
	case d.queue <- msg:
		return nil
	case <-d.ctx.Done():
		return ErrDispatcherStopped
	default:
		return ErrQueueFull
	}
}

// Start starts the dispatcher workers
func (d *Dispatcher) Start() {
	for i := 0; i < d.workers; i++ {
		d.wg.Add(1)
		go d.worker(i)
	}
}

// Stop stops the dispatcher
func (d *Dispatcher) Stop() {
	d.cancel()
	d.wg.Wait()
}

// worker processes messages from the queue
func (d *Dispatcher) worker(id int) {
	defer d.wg.Done()

	d.log.Info("Dispatcher worker started", logger.Int("worker_id", id))

	for {
		select {
		case <-d.ctx.Done():
			d.log.Info("Dispatcher worker stopped", logger.Int("worker_id", id))
			return
		case msg := <-d.queue:
			d.handleMessage(msg)
		}
	}
}

// handleMessage handles a message
func (d *Dispatcher) handleMessage(msg *Message) {
	d.handlersMu.RLock()
	handler, exists := d.handlers[msg.Target]
	d.handlersMu.RUnlock()

	if !exists {
		d.log.Warn("No handler for target", logger.String("target", msg.Target))
		return
	}

	if err := handler(d.ctx, msg); err != nil {
		d.log.Error("Handler error",
			logger.String("target", msg.Target),
			logger.Error(err),
		)
	}
}
