package writer

import (
	"bytes"
	"sync"
	"sync/atomic"
)

// 定义不同大小的缓冲区类别
// Define different buffer size categories
const (
	// 小缓冲区大小 (<= 1KB)
	// Small buffer size (<= 1KB)
	smallBufferSize = 1024

	// 中等缓冲区大小 (<= 8KB)
	// Medium buffer size (<= 8KB)
	mediumBufferSize = 8 * 1024

	// 大缓冲区大小 (<= 32KB)
	// Large buffer size (<= 32KB)
	largeBufferSize = 32 * 1024
)

// BufferSizeStats 缓冲区大小统计
// BufferSizeStats buffer size statistics
type BufferSizeStats struct {
	smallCount  atomic.Int64 // 小缓冲区计数
	mediumCount atomic.Int64 // 中等缓冲区计数
	largeCount  atomic.Int64 // 大缓冲区计数
	overSize    atomic.Int64 // 超大缓冲区计数
	totalCalls  atomic.Int64 // 总调用次数
}

// BufferPool 是一个结构体，它包含多个同步池以支持不同大小的缓冲区
// BufferPool is a struct that contains multiple sync pools to support buffers of different sizes
type BufferPool struct {
	smallPool  *sync.Pool      // 小缓冲区池（<= 1KB）
	mediumPool *sync.Pool      // 中等缓冲区池（<= 8KB）
	largePool  *sync.Pool      // 大缓冲区池（<= 32KB）
	stats      BufferSizeStats // 大小统计信息
}

// NewBufferPool 是一个函数，它创建并返回一个新的 BufferPool
// NewBufferPool is a function that creates and returns a new BufferPool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		// 创建小缓冲区池
		// Create small buffer pool
		smallPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, smallBufferSize))
			},
		},

		// 创建中等缓冲区池
		// Create medium buffer pool
		mediumPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, mediumBufferSize))
			},
		},

		// 创建大缓冲区池
		// Create large buffer pool
		largePool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, largeBufferSize))
			},
		},

		// 初始化统计信息
		// Initialize statistics
		stats: BufferSizeStats{},
	}
}

// Get 是一个方法，它从 BufferPool 获取一个合适大小的缓冲区
// 如果指定了预期大小，会尝试返回合适容量的缓冲区
// Get is a method that gets an appropriately sized buffer from the BufferPool
// If an expected size is specified, it will try to return a buffer with suitable capacity
func (p *BufferPool) Get() *bytes.Buffer {
	// 首先尝试获取小缓冲区，这是最常见的情况
	// First try to get a small buffer, which is the most common case
	return p.getWithHint(0)
}

// getWithHint 根据大小提示获取适当的缓冲区
// getWithHint gets an appropriate buffer based on size hint
func (p *BufferPool) getWithHint(sizeHint int) *bytes.Buffer {
	// 更新统计信息
	// Update statistics
	p.stats.totalCalls.Add(1)

	// 根据大小提示选择合适的缓冲区池
	// Choose the appropriate buffer pool based on the size hint
	if sizeHint <= smallBufferSize {
		p.stats.smallCount.Add(1)
		return p.smallPool.Get().(*bytes.Buffer)
	} else if sizeHint <= mediumBufferSize {
		p.stats.mediumCount.Add(1)
		return p.mediumPool.Get().(*bytes.Buffer)
	} else if sizeHint <= largeBufferSize {
		p.stats.largeCount.Add(1)
		return p.largePool.Get().(*bytes.Buffer)
	} else {
		// 对于超大缓冲区，直接创建新的，不放入池中
		// For oversized buffers, create a new one directly, don't put it in the pool
		p.stats.overSize.Add(1)
		return bytes.NewBuffer(make([]byte, 0, sizeHint))
	}
}

// Put 是一个方法，它将一个缓冲区归还到 BufferPool 中
// Put is a method that puts a buffer back into the BufferPool
func (p *BufferPool) Put(e *bytes.Buffer) {
	// 如果缓冲区为空，则直接返回
	// If the buffer is nil, return directly
	if e == nil {
		return
	}

	// 重置缓冲区
	// Reset the buffer
	e.Reset()

	// 根据缓冲区容量决定放入哪个池
	// Decide which pool to put the buffer into based on its capacity
	cap := e.Cap()
	if cap <= smallBufferSize {
		p.smallPool.Put(e)
	} else if cap <= mediumBufferSize {
		p.mediumPool.Put(e)
	} else if cap <= largeBufferSize {
		p.largePool.Put(e)
	}
	// 超大缓冲区不放回池中，让GC回收
	// Oversized buffers are not put back into the pool, let GC reclaim them
}

// GetStats 返回缓冲池统计信息
// GetStats returns buffer pool statistics
func (p *BufferPool) GetStats() (small, medium, large, oversize, total int64) {
	small = p.stats.smallCount.Load()
	medium = p.stats.mediumCount.Load()
	large = p.stats.largeCount.Load()
	oversize = p.stats.overSize.Load()
	total = p.stats.totalCalls.Load()
	return
}
