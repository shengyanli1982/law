package queue

import (
	"bytes"
	"fmt"
)

// BufferQueue 是面向内部写入路径的兼容队列。
// 对外暴露 interface{} Push/Pop，内部使用 *bytes.Buffer 泛型队列。
type BufferQueue struct {
	queue *MPSCQueue[*bytes.Buffer]
}

// NewBufferQueue 创建一个用于 *bytes.Buffer 的兼容队列。
func NewBufferQueue() *BufferQueue {
	return &BufferQueue{queue: NewMPSCQueue[*bytes.Buffer]()}
}

// NewBufferQueueWithLimits 创建带上限的 *bytes.Buffer 兼容队列。
func NewBufferQueueWithLimits(maxItems int, maxBytes int64) *BufferQueue {
	return &BufferQueue{queue: NewMPSCQueueWithLimits[*bytes.Buffer](maxItems, maxBytes)}
}

// Push 兼容旧 Queue 接口。
func (q *BufferQueue) Push(value interface{}) {
	if value == nil {
		return
	}

	buffer, ok := value.(*bytes.Buffer)
	if !ok {
		panic(fmt.Sprintf("law: queue expects *bytes.Buffer, got %T", value))
	}
	q.queue.Push(buffer)
}

// Pop 兼容旧 Queue 接口。
func (q *BufferQueue) Pop() interface{} {
	return q.queue.Pop()
}

// PushBuffer 类型安全入队。
func (q *BufferQueue) PushBuffer(value *bytes.Buffer) {
	if value == nil {
		return
	}
	q.queue.Push(value)
}

// PopBuffer 类型安全出队。
func (q *BufferQueue) PopBuffer() *bytes.Buffer {
	return q.queue.Pop()
}

// Len 返回当前队列长度。
func (q *BufferQueue) Len() int {
	return q.queue.Len()
}
