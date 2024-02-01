package queue

import (
	"sync/atomic"
	"unsafe"
)

type Element struct {
	next  unsafe.Pointer
	value interface{}
}

func LoadItem(p *unsafe.Pointer) *Element {
	return (*Element)(atomic.LoadPointer(p))
}

func CasItem(p *unsafe.Pointer, old, new *Element) bool {
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}

type LockFreeQueue struct {
	head   unsafe.Pointer
	tail   unsafe.Pointer
	length uint64
}

func NewLockFreeQueue() *LockFreeQueue {
	head := Element{next: nil, value: nil}
	return &LockFreeQueue{
		tail:   unsafe.Pointer(&head),
		head:   unsafe.Pointer(&head),
		length: 0,
	}
}

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

func (q *LockFreeQueue) Length() uint64 {
	return atomic.LoadUint64(&q.length)
}

func (q *LockFreeQueue) Reset() {
	q.head = nil
	q.tail = nil
	q.length = 0
}
