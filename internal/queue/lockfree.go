package queue

import (
	"sync/atomic"
	"unsafe"
)

// Element is an element of the queue.
type Element struct {
	// next is the next pointer.
	next unsafe.Pointer

	// value is the value of the element.
	value interface{}
}

// LoadItem loads the element from the given pointer.
func LoadItem(p *unsafe.Pointer) *Element {
	return (*Element)(atomic.LoadPointer(p))
}

// CasItem compares and swaps the element in the given pointer.
func CasItem(p *unsafe.Pointer, old, new *Element) bool {
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}

type LockFreeQueue struct {
	head   unsafe.Pointer
	tail   unsafe.Pointer
	length uint64
}

// NewLockFreeQueue creates a new lock-free queue.
func NewLockFreeQueue() *LockFreeQueue {
	head := Element{next: nil, value: nil}
	return &LockFreeQueue{
		tail:   unsafe.Pointer(&head),
		head:   unsafe.Pointer(&head),
		length: 0,
	}
}

// Push adds a new element at the tail of the queue.
func (q *LockFreeQueue) Push(value interface{}) {
	newItem := &Element{next: nil, value: value}
	for {
		tail := LoadItem(&q.tail)
		lastNext := LoadItem(&tail.next)
		if tail == LoadItem(&q.tail) {
			if lastNext == nil {
				if CasItem(&tail.next, lastNext, newItem) {
					CasItem(&q.tail, tail, newItem)
					atomic.AddUint64(&q.length, 1)
					return
				}
			} else {
				CasItem(&q.tail, tail, lastNext)
			}
		}
	}
}

// Pop pop and returns the first element from the queue.
func (q *LockFreeQueue) Pop() interface{} {
	for {
		head := LoadItem(&q.head)
		tail := LoadItem(&q.tail)
		firstNext := LoadItem(&head.next)
		if head == LoadItem(&q.head) {
			if head == tail {
				if firstNext == nil {
					return nil
				}
				CasItem(&q.tail, tail, firstNext)
			} else {
				v := firstNext.value
				if CasItem(&q.head, head, firstNext) {
					atomic.AddUint64(&q.length, ^uint64(0))
					return v
				}
			}
		}
	}
}

// Length returns the current length of the queue.
func (q *LockFreeQueue) Length() uint64 {
	return atomic.LoadUint64(&q.length)
}

// Reset resets the queue.
func (q *LockFreeQueue) Reset() {
	q.head = nil
	q.tail = nil
	q.length = 0
}
