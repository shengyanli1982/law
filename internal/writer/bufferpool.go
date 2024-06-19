package writer

import (
	"bytes"
	"sync"
)

// BufferPool 是一个结构体，它包含一个同步池
// BufferPool is a struct that contains a sync pool
type BufferPool struct {
	pool *sync.Pool
}

// NewBufferPool 是一个函数，它创建并返回一个新的 elementPool
// NewBufferPool is a function that creates and returns a new elementPool
func NewBufferPool() *BufferPool {
	// 创建一个新的同步池
	// Create a new sync pool
	pool := &sync.Pool{
		// New 是一个函数，它创建并返回一个新的 Element
		// New is a function that creates and returns a new Element
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 1024))
		},
	}

	// 返回一个新的 elementPool，它包含刚刚创建的同步池
	// Return a new elementPool that contains the sync pool we just created
	return &BufferPool{pool: pool}
}

// Get 是一个方法，它从 elementPool 的同步池中获取一个 Element
// Get is a method that gets an Element from the sync pool of the elementPool
func (p *BufferPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

// Put 是一个方法，它将一个 Element 放回 elementPool 的同步池中
// Put is a method that puts an Element back into the sync pool of the elementPool
func (p *BufferPool) Put(e *bytes.Buffer) {
	// 如果 Element 不为空，则重置它并将其放回同步池中
	// If the Element is not nil, reset it and put it back into the sync pool
	if e != nil {
		e.Reset()
		p.pool.Put(e)
	}
}
