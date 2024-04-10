package lockfree

import (
	"sync/atomic"

	"unsafe"
)

type LockFreeStack struct {
	top unsafe.Pointer

	length uint64
}

func NewLockFreeStack() *LockFreeStack {

	dummy := NewNode()

	return &LockFreeStack{

		top: unsafe.Pointer(dummy),
	}

}

func (s *LockFreeStack) Push(value interface{}) {

	node := NewNode()

	node.value = value

	for {

		top := loadNode(&s.top)

		node.next = unsafe.Pointer(top)

		if compareAndSwapNode(&s.top, top, node) {

			atomic.AddUint64(&s.length, 1)

			return

		}

	}

}

func (s *LockFreeStack) Pop() interface{} {

	for {

		top := loadNode(&s.top)

		if top == nil {

			return nil

		}

		next := loadNode(&top.next)

		if compareAndSwapNode(&s.top, top, next) {

			atomic.AddUint64(&s.length, ^uint64(0))

			v := top.value

			top.Reset()

			return v

		}

	}

}

func (s *LockFreeStack) Length() uint64 {

	return atomic.LoadUint64(&s.length)

}

func (s *LockFreeStack) Reset() {

	dummy := NewNode()

	s.top = unsafe.Pointer(dummy)

	atomic.StoreUint64(&s.length, 0)

}
