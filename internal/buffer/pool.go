package buffer

import "sync"

type ExtraBufferPool struct {
	bp sync.Pool // 同步池 (sync pool)
}

func NewExtraBufferPool(size int) *ExtraBufferPool {
	if size <= 0 {
		size = DefaultBufferSize // 如果大小小于等于0，则使用默认缓冲区大小 (if size is less than or equal to 0, use default buffer size)
	}
	return &ExtraBufferPool{
		bp: sync.Pool{
			New: func() any {
				return NewExtraBuffer(size) // 创建新的ExtraBuffer对象 (create a new ExtraBuffer object)
			},
		},
	}
}

func (p *ExtraBufferPool) Get() *ExtraBuffer {
	return p.bp.Get().(*ExtraBuffer) // 从池中获取ExtraBuffer对象 (get ExtraBuffer object from the pool)
}

func (p *ExtraBufferPool) Put(b *ExtraBuffer) {
	if b != nil {
		b.updateAt = 0 // 重置更新时间 (reset update time)
		b.buff.Reset() // 重置字节缓冲区 (reset byte buffer)
		p.bp.Put(b)    // 将ExtraBuffer对象放回池中 (put ExtraBuffer object back into the pool)
	}
}
