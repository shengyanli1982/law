package lockfree

import (
	"sync/atomic"

	"unsafe"
)

type LockFreeQueue struct {
	length uint64

	head unsafe.Pointer

	tail unsafe.Pointer
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

					result := first.value

					head.Reset()

					return result

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
