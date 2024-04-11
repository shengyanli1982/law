package law

import "sync"

// Element 是一个结构体，包含一个字节缓冲区和一个更新时间戳
// Element is a structure that contains a byte buffer and an update timestamp
type Element struct {
	// buffer 是一个字节缓冲区
	// buffer is a byte buffer
	buffer []byte

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

// Reset 是 Element 结构体的一个方法，用于重置 Element 的状态
// Reset is a method of the Element structure, used to reset the state of the Element
func (e *Element) Reset() {
	// 清空字节缓冲区
	// Empty the byte buffer
	e.buffer = nil

	// 重置更新时间戳
	// Reset the update timestamp
	e.updateAt = 0

}

// ElementPool 是一个结构体，包含一个同步池
// ElementPool is a structure that contains a sync pool
type ElementPool struct {
	// pool 是一个同步池，用于存储 Element 实例
	// pool is a sync pool for storing Element instances
	pool *sync.Pool
}

// NewElementPool 是一个构造函数，用于创建一个新的 ElementPool 实例。
// NewElementPool is a constructor function for creating a new ElementPool instance.
func NewElementPool() *ElementPool {
	// 返回一个新的 ElementPool 实例，其中包含一个同步池。
	// Return a new ElementPool instance, which contains a sync.Pool.
	return &ElementPool{
		// 初始化一个 sync.Pool。
		// Initialize a sync.Pool.
		pool: &sync.Pool{
			// sync.Pool 的 New 方法用于创建新的 Element 实例。
			// The New method of sync.Pool is used to create new Element instances.
			New: func() interface{} {
				// 调用 NewElement 函数来创建一个新的 Element 实例并返回。
				// Call the NewElement function to create a new Element instance and return it.
				return NewElement()
			},
		},
	}
}

// Get 是 ElementPool 结构体的一个方法，用于从同步池中获取一个 Element 实例
// Get is a method of the ElementPool structure, used to get an Element instance from the sync pool
func (ep *ElementPool) Get() *Element {
	// 从同步池中获取一个 Element 实例
	// Get an Element instance from the sync pool
	return ep.pool.Get().(*Element)

}

// Put 是 ElementPool 结构体的一个方法，用于将一个 Element 实例放回同步池
// Put is a method of the ElementPool structure, used to put an Element instance back into the sync pool
func (ep *ElementPool) Put(e *Element) {
	// 如果 Element 实例不为空
	// If the Element instance is not null
	if e != nil {

		// 重置 Element 实例的状态
		// Reset the state of the Element instance
		e.Reset()

		// 将 Element 实例放回同步池
		// Put the Element instance back into the sync pool
		ep.pool.Put(e)
	}
}
