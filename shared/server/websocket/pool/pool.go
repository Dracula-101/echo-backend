package pool

import (
	"sync"
)

// Pool is a generic object pool
type Pool struct {
	pool sync.Pool
}

// NewPool creates a new pool
func NewPool(newFunc func() interface{}) *Pool {
	return &Pool{
		pool: sync.Pool{
			New: newFunc,
		},
	}
}

// Get gets an object from the pool
func (p *Pool) Get() interface{} {
	return p.pool.Get()
}

// Put returns an object to the pool
func (p *Pool) Put(obj interface{}) {
	p.pool.Put(obj)
}

// ByteBufferPool is a pool of byte buffers
type ByteBufferPool struct {
	pool *Pool
	size int
}

// NewByteBufferPool creates a new byte buffer pool
func NewByteBufferPool(size int) *ByteBufferPool {
	return &ByteBufferPool{
		pool: NewPool(func() interface{} {
			return make([]byte, size)
		}),
		size: size,
	}
}

// Get gets a buffer from the pool
func (p *ByteBufferPool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (p *ByteBufferPool) Put(buf []byte) {
	if len(buf) == p.size {
		p.pool.Put(buf)
	}
}
