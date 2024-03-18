package stack

import (
	"sync/atomic"
	"unsafe"
)

// Element 是栈中的元素
// Element is an element in the stack
type Element struct {
	next  unsafe.Pointer // 指向下一个元素的指针，Points to the next element
	value interface{}    // 元素的值，The value of the element
}

// LockFreeStack 是一个无锁栈
// LockFreeStack is a lock-free stack
type LockFreeStack struct {
	top    unsafe.Pointer // 栈顶指针，Pointer to the top of the stack
	length uint64         // 栈的长度，The length of the stack
}

// NewLockFreeStack 创建一个新的无锁栈
// NewLockFreeStack creates a new lock-free stack
func NewLockFreeStack() *LockFreeStack {
	return &LockFreeStack{
		// 初始化栈顶指针为空元素
		// Initialize the top pointer to an empty element
		top: unsafe.Pointer(&Element{}),
	}
}

// Pop 从栈中弹出一个元素
// Pop pops an element from the stack
func (s *LockFreeStack) Pop() interface{} {
	for {
		// 获取栈顶指针
		// Get the top pointer
		top := atomic.LoadPointer(&s.top)

		// 栈为空，返回nil
		// If the stack is empty, return nil
		if top == nil {
			return nil
		}

		// 将栈顶指针转换为Element类型
		// Convert the top pointer to Element type
		item := (*Element)(top)

		// 获取下一个元素的指针
		// Get the pointer of the next element
		next := atomic.LoadPointer(&item.next)

		// 将栈顶指针指向下一个元素
		// Set the top pointer to the next element
		if atomic.CompareAndSwapPointer(&s.top, top, next) {
			// 栈长度减1
			// Decrement the length of the stack
			atomic.AddUint64(&s.length, ^uint64(0))

			// 获取弹出元素的值
			// Get the value of the popped element
			value := item.value

			// 清除引用，帮助GC
			// Clear the reference to help GC
			item.next = nil
			item.value = nil

			// 返回弹出元素的值
			// Return the value of the popped element
			return value
		}
	}
}

// Push 将一个元素压入栈中
// Push pushes an element into the stack
func (s *LockFreeStack) Push(value interface{}) {
	// 创建一个新的元素
	// Create a new element
	item := &Element{value: value}

	// 无限循环，直到压入元素成功
	// Infinite loop until the element is pushed successfully
	for {
		// 获取栈顶指针
		// Get the top pointer
		top := atomic.LoadPointer(&s.top)

		// 将新元素的next指针指向原来的栈顶元素
		// Set the next pointer of the new element to the original top element
		item.next = top

		// 将栈顶指针指向新元素
		// Set the top pointer to the new element
		if atomic.CompareAndSwapPointer(&s.top, top, unsafe.Pointer(item)) {
			// 栈长度加1
			// Increment the length of the stack
			atomic.AddUint64(&s.length, 1)

			return
		}
	}
}

// Len 返回栈的长度
// Len returns the length of the stack
func (s *LockFreeStack) Len() uint64 {
	// 使用原子操作获取栈的长度
	// Use atomic operation to get the length of the stack
	return atomic.LoadUint64(&s.length)
}

// Reset 重置栈
// Reset resets the stack
func (s *LockFreeStack) Reset() {
	// 将栈顶指针指向空元素
	// Set the top pointer to an empty element
	s.top = unsafe.Pointer(&Element{})

	// 使用原子操作将栈长度置为0
	// Use atomic operation to set the length of the stack to 0
	atomic.StoreUint64(&s.length, 0)
}
