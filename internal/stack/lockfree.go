package stack

import (
	"sync/atomic"
	"unsafe"
)

type Element struct {
	next  unsafe.Pointer
	value interface{}
}

type LockFreeStack struct {
	top    unsafe.Pointer
	length uint64
}

func NewLockFreeStack() *LockFreeStack {
	return &LockFreeStack{
		top: unsafe.Pointer(&Element{}),
	}
}

func (s *LockFreeStack) Pop() interface{} {
	for {
		top := atomic.LoadPointer(&s.top)
		if top == nil {
			return nil
		}
		item := (*Element)(top)
		next := atomic.LoadPointer(&item.next)
		if atomic.CompareAndSwapPointer(&s.top, top, next) {
			atomic.AddUint64(&s.length, ^uint64(0))
			value := item.value
			item.next = nil // 清除引用，帮助 GC
			item.value = nil
			return value
		}
	}
}

func (s *LockFreeStack) Push(value interface{}) {
	item := &Element{value: value}
	var top unsafe.Pointer
	for {
		top = atomic.LoadPointer(&s.top)
		item.next = top
		if atomic.CompareAndSwapPointer(&s.top, top, unsafe.Pointer(item)) {
			atomic.AddUint64(&s.length, 1)
			return
		}
	}
}

func (s *LockFreeStack) Len() uint64 {
	return atomic.LoadUint64(&s.length)
}

func (s *LockFreeStack) Reset() {
	s.top = unsafe.Pointer(&Element{})
	atomic.StoreUint64(&s.length, 0)
}
