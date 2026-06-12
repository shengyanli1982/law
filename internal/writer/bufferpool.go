package writer

import (
	"bytes"
	"runtime"
	"sync"
)

// 定义不同大小的缓冲区类别
const (
	// 超小缓冲区大小 (<= 128B)
	tinyBufferSize = 128

	// 小缓冲区大小 (<= 1KB)
	smallBufferSize = 1024

	// 中等缓冲区大小 (<= 8KB)
	mediumBufferSize = 8 * 1024

	// 大缓冲区大小 (<= 32KB)
	largeBufferSize = 32 * 1024
)

// BufferPool 是一个结构体，它包含多个同步池以支持不同大小的缓冲区
type BufferPool struct {
	tinyPool   *sync.Pool // 超小缓冲区池（<= 128B）
	smallPool  *sync.Pool // 小缓冲区池（<= 1KB）
	mediumPool *sync.Pool // 中等缓冲区池（<= 8KB）
	largePool  *sync.Pool // 大缓冲区池（<= 32KB）
}

// NewBufferPool 是一个函数，它创建并返回一个新的 BufferPool
func NewBufferPool() *BufferPool {
	p := &BufferPool{
		tinyPool: &sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, tinyBufferSize))
			},
		},
		smallPool: &sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, smallBufferSize))
			},
		},
		mediumPool: &sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, mediumBufferSize))
			},
		},
		largePool: &sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, largeBufferSize))
			},
		},
	}

	warmUpCount := runtime.GOMAXPROCS(0) * 4
	if warmUpCount < 8 {
		warmUpCount = 8
	}
	for i := 0; i < warmUpCount; i++ {
		p.tinyPool.Put(bytes.NewBuffer(make([]byte, 0, tinyBufferSize)))
		p.smallPool.Put(bytes.NewBuffer(make([]byte, 0, smallBufferSize)))
		p.mediumPool.Put(bytes.NewBuffer(make([]byte, 0, mediumBufferSize)))
		p.largePool.Put(bytes.NewBuffer(make([]byte, 0, largeBufferSize)))
	}

	return p
}

// Get 是一个方法，它从 BufferPool 获取一个合适大小的缓冲区
// 如果指定了预期大小，会尝试返回合适容量的缓冲区
func (p *BufferPool) Get() *bytes.Buffer {
	return p.GetWithHint(0)
}

// GetWithHint 根据大小提示获取适当的缓冲区
func (p *BufferPool) GetWithHint(sizeHint int) *bytes.Buffer {
	// 根据大小提示选择合适的缓冲区池
	if sizeHint <= tinyBufferSize {
		return p.tinyPool.Get().(*bytes.Buffer)
	} else if sizeHint <= smallBufferSize {
		return p.smallPool.Get().(*bytes.Buffer)
	} else if sizeHint <= mediumBufferSize {
		return p.mediumPool.Get().(*bytes.Buffer)
	} else if sizeHint <= largeBufferSize {
		return p.largePool.Get().(*bytes.Buffer)
	} else {
		// 对于超大缓冲区，直接创建新的，不放入池中
		return bytes.NewBuffer(make([]byte, 0, sizeHint))
	}
}

// Put 是一个方法，它将一个缓冲区归还到 BufferPool 中
func (p *BufferPool) Put(e *bytes.Buffer) {
	// 如果缓冲区为空，则直接返回
	if e == nil {
		return
	}

	// 重置缓冲区
	e.Reset()

	// 按 buffer 实际容量路由到匹配的池。
	// buffer 经 Grow 扩容后 cap 增大，会被放入更大的池，
	// 这是合理行为：该 buffer 在更大的池中仍可被后续需要相似大小的请求复用。
	cap := e.Cap()
	if cap <= tinyBufferSize {
		p.tinyPool.Put(e)
	} else if cap <= smallBufferSize {
		p.smallPool.Put(e)
	} else if cap <= mediumBufferSize {
		p.mediumPool.Put(e)
	} else if cap <= largeBufferSize {
		p.largePool.Put(e)
	}
	// cap > largeBufferSize → 丢弃，让 GC 回收
}
