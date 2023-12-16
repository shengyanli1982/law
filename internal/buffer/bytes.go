package buffer

import (
	"bytes"
	"sync"
)

const defaultBufferSize = 2048 // 默认缓冲区大小 (default buffer size)

type ExtraBuffer struct {
	buff     *bytes.Buffer // 字节缓冲区 (byte buffer)
	updateAt int64         // 更新时间 (update time)
}

func NewExtraBuffer(size int) *ExtraBuffer {
	return &ExtraBuffer{
		buff:     bytes.NewBuffer(make([]byte, 0, size)), // 创建指定大小的字节缓冲区 (create a byte buffer with specified size)
		updateAt: 0,                                      // 初始化更新时间为0 (initialize update time to 0)
	}
}

func (b *ExtraBuffer) Buffer() *bytes.Buffer {
	return b.buff // 返回字节缓冲区 (return byte buffer)
}

func (b *ExtraBuffer) UpdateAt() int64 {
	return b.updateAt // 返回更新时间 (return update time)
}

func (b *ExtraBuffer) SetUpdateAt(updateAt int64) {
	b.updateAt = updateAt // 设置更新时间 (set update time)
}

type ExtraBufferPool struct {
	bp sync.Pool // 同步池 (sync pool)
}

func NewExtraBufferPool(size int) *ExtraBufferPool {
	if size <= 0 {
		size = defaultBufferSize // 如果大小小于等于0，则使用默认缓冲区大小 (if size is less than or equal to 0, use default buffer size)
	}
	return &ExtraBufferPool{
		bp: sync.Pool{
			New: func() interface{} {
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
