package queue

import (
	"bytes"
	"sync"
)

type queueNode[T any] struct {
	next  *queueNode[T]
	value T
	size  int
}

// MPSCQueue 是基于 mutex/cond 的链表队列，并使用 sync.Pool 复用节点。
// 该实现面向多生产者、单消费者场景，默认不丢数据。
type MPSCQueue[T any] struct {
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond

	head *queueNode[T]
	tail *queueNode[T]

	count int
	bytes int64

	maxItems int
	maxBytes int64

	nodePool sync.Pool
}

// NewMPSCQueue 创建一个无界泛型队列。
func NewMPSCQueue[T any]() *MPSCQueue[T] {
	q := &MPSCQueue[T]{}
	q.notEmpty = sync.NewCond(&q.mu)
	q.notFull = sync.NewCond(&q.mu)
	q.nodePool.New = func() any {
		return &queueNode[T]{}
	}
	return q
}

// NewMPSCQueueWithLimits 创建带上限的泛型队列。
// 达到上限后 Push 会阻塞等待可用空间。
// maxItems <= 0 表示不限条数，maxBytes <= 0 表示不限字节数。
func NewMPSCQueueWithLimits[T any](maxItems int, maxBytes int64) *MPSCQueue[T] {
	q := NewMPSCQueue[T]()
	q.maxItems = maxItems
	q.maxBytes = maxBytes
	return q
}

func estimateSize[T any](value T) int {
	switch v := any(value).(type) {
	case *bytes.Buffer:
		return v.Len()
	case []byte:
		return len(v)
	case string:
		return len(v)
	case interface{ Len() int }:
		return v.Len()
	default:
		return 1
	}
}

func (q *MPSCQueue[T]) isFull(nextSize int) bool {
	if q.maxItems > 0 && q.count >= q.maxItems {
		return true
	}
	if q.maxBytes > 0 && q.bytes+int64(nextSize) > q.maxBytes {
		return true
	}
	return false
}

// Push 将值入队。
// 当配置了上限且队列满时，会阻塞等待空间。
func (q *MPSCQueue[T]) Push(value T) {
	if any(value) == nil {
		return
	}

	size := estimateSize(value)

	q.mu.Lock()
	for q.isFull(size) {
		q.notFull.Wait()
	}

	node := q.nodePool.Get().(*queueNode[T])
	node.value = value
	node.size = size
	node.next = nil

	if q.tail == nil {
		q.head = node
		q.tail = node
	} else {
		q.tail.next = node
		q.tail = node
	}

	q.count++
	q.bytes += int64(size)
	q.notEmpty.Signal()
	q.mu.Unlock()
}

// Pop 出队一个值；队列为空时返回 T 的零值。
func (q *MPSCQueue[T]) Pop() T {
	var zero T

	q.mu.Lock()
	node := q.head
	if node == nil {
		q.mu.Unlock()
		return zero
	}

	q.head = node.next
	if q.head == nil {
		q.tail = nil
	}
	q.count--
	q.bytes -= int64(node.size)
	if q.maxItems > 0 || q.maxBytes > 0 {
		q.notFull.Signal()
	}
	q.mu.Unlock()

	value := node.value
	var resetValue T
	node.value = resetValue
	node.next = nil
	node.size = 0
	q.nodePool.Put(node)
	return value
}

// Len 返回当前队列中的元素数量。
func (q *MPSCQueue[T]) Len() int {
	q.mu.Lock()
	n := q.count
	q.mu.Unlock()
	return n
}
