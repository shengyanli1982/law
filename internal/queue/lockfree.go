package queue

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type Node struct {
	value interface{}
	next  unsafe.Pointer
}

func NewNode() *Node {
	return &Node{}
}

func (n *Node) Reset() {
	n.value = nil
	n.next = nil
}

type NodePool struct {
	pool *sync.Pool
}

func loadNode(p *unsafe.Pointer) *Node {
	return (*Node)(atomic.LoadPointer(p))
}

func compareAndSwapNode(p *unsafe.Pointer, old, new *Node) bool {
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}

type LockFreeQueue struct {
	length uint64
	head   unsafe.Pointer
	tail   unsafe.Pointer
}

func NewLockFreeQueue() *LockFreeQueue {
	dummy := NewNode()
	return &LockFreeQueue{
		head: unsafe.Pointer(dummy),
		tail: unsafe.Pointer(dummy),
	}
}

func (q *LockFreeQueue) Push(value interface{}) {
	node := NewNode()
	node.value = value

	for {
		tail := loadNode(&q.tail)
		next := loadNode(&tail.next)

		if tail == loadNode(&q.tail) {
			if next == nil {
				if compareAndSwapNode(&tail.next, next, node) {
					compareAndSwapNode(&q.tail, tail, node)
					atomic.AddUint64(&q.length, 1)
					return
				}
			} else {
				compareAndSwapNode(&q.tail, tail, next)
			}
		}
	}
}

func (q *LockFreeQueue) Pop() interface{} {
	for {
		head := loadNode(&q.head)
		tail := loadNode(&q.tail)
		first := loadNode(&head.next)

		if head == loadNode(&q.head) {
			if head == tail {
				if first == nil {
					return nil
				}
				compareAndSwapNode(&q.tail, tail, first)
			} else {
				if compareAndSwapNode(&q.head, head, first) {
					atomic.AddUint64(&q.length, ^uint64(0))
					return first.value
				}
			}
		}
	}
}

func (q *LockFreeQueue) Length() uint64 {
	return atomic.LoadUint64(&q.length)
}

func (q *LockFreeQueue) Reset() {
	dummy := NewNode()
	q.head = unsafe.Pointer(dummy)
	q.tail = unsafe.Pointer(dummy)
	atomic.StoreUint64(&q.length, 0)
}
