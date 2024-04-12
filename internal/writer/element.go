package writer

import (
	"bytes"
	"sync"
)

// Element 是一个结构体，包含一个字节缓冲区和一个更新时间戳
// Element is a structure that contains a byte buffer and an update timestamp
type Element struct {
	// buffer 是一个字节缓冲区
	// buffer is a byte buffer
	buffer bytes.Buffer

	// updateAt 是一个更新时间戳
	// updateAt is an update timestamp
	updateAt int64
}

// NewElement 是一个构造函数，用于创建一个新的 Element 实例
// NewElement is a constructor function for creating a new Element instance
func NewElement() *Element {
	// 返回一个新的 Element 实例
	// Return a new Element instance
	return &Element{}
}

func (e *Element) GetBuffer() *bytes.Buffer {
	return &e.buffer
}

func (e *Element) GetUpdateAt() int64 {
	return e.updateAt
}

func (e *Element) SetUpdateAt(updateAt int64) {
	e.updateAt = updateAt
}

// Reset 是 Element 结构体的一个方法，用于重置 Element 的状态
// Reset is a method of the Element structure, used to reset the state of the Element
func (e *Element) Reset() {
	// 清空字节缓冲区
	// Empty the byte buffer
	e.buffer.Reset()

	// 重置更新时间戳
	// Reset the update timestamp
	e.updateAt = 0
}

// ElementPool 是一个结构体，它包含一个同步池
// ElementPool is a struct that contains a sync pool
type ElementPool struct {
	pool *sync.Pool
}

// NewElementPool 是一个函数，它创建并返回一个新的 elementPool
// NewElementPool is a function that creates and returns a new elementPool
func NewElementPool() *ElementPool {
	// 创建一个新的同步池
	// Create a new sync pool
	pool := &sync.Pool{
		// New 是一个函数，它创建并返回一个新的 Element
		// New is a function that creates and returns a new Element
		New: func() interface{} {
			return NewElement()
		},
	}

	// 返回一个新的 elementPool，它包含刚刚创建的同步池
	// Return a new elementPool that contains the sync pool we just created
	return &ElementPool{pool: pool}
}

// Get 是一个方法，它从 elementPool 的同步池中获取一个 Element
// Get is a method that gets an Element from the sync pool of the elementPool
func (p *ElementPool) Get() *Element {
	return p.pool.Get().(*Element)
}

// Put 是一个方法，它将一个 Element 放回 elementPool 的同步池中
// Put is a method that puts an Element back into the sync pool of the elementPool
func (p *ElementPool) Put(e *Element) {
	// 如果 Element 不为空，则重置它并将其放回同步池中
	// If the Element is not nil, reset it and put it back into the sync pool
	if e != nil {
		e.Reset()
		p.pool.Put(e)
	}
}
