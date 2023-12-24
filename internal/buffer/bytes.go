package buffer

import (
	"bytes"
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
