package queue

import (
	"sync/atomic"
	"unsafe"
)

// Element 是无锁队列中的元素
// Element represents an element in the lock-free queue.
type Element struct {
	next  unsafe.Pointer // next 是指向下一个元素的指针
	value interface{}    // value 是元素的值
}

// LoadItem 加载指针中的元素
// LoadItem loads the element from the given pointer.
func LoadItem(p *unsafe.Pointer) *Element {
	// 使用原子操作加载指针中的元素
	// Use atomic operation to load the element from the pointer
	return (*Element)(atomic.LoadPointer(p))
}

// CasItem 比较并交换给定指针中的元素
// CasItem compares and swaps the element in the given pointer.
func CasItem(p *unsafe.Pointer, old, new *Element) bool {
	// 使用原子操作比较并交换指针中的元素
	// Use atomic operation to compare and swap the element in the pointer
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}

// LockFreeQueue 是一个无锁队列
// LockFreeQueue represents a lock-free queue.
type LockFreeQueue struct {
	head   unsafe.Pointer // head 是指向队列头部的指针
	tail   unsafe.Pointer // tail 是指向队列尾部的指针
	length uint64         // length 是队列的长度
}

// NewLockFreeQueue 创建一个新的无锁队列
// NewLockFreeQueue creates a new lock-free queue.
func NewLockFreeQueue() *LockFreeQueue {
	head := Element{} // 创建一个新的元素作为头部
	return &LockFreeQueue{
		tail: unsafe.Pointer(&head), // 设置尾部指针指向头部元素
		head: unsafe.Pointer(&head), // 设置头部指针指向头部元素
	}
}

// Push 将一个新元素添加到队列中
// Push adds a new element to the queue.
func (q *LockFreeQueue) Push(value interface{}) {
	// 创建一个新的元素
	// Create a new element
	newItem := &Element{next: nil, value: value}

	// 将新元素添加到队列中
	// Add the new element to the queue
	for {
		// 获取队列尾部元素
		// Get the tail element of the queue
		tail := LoadItem(&q.tail)

		// 获取队列尾部元素的下一个元素
		// Get the next element of the tail element
		lastNext := LoadItem(&tail.next)

		// 如果队列尾部元素没有下一个元素，就将新元素添加到队列尾部
		// If the tail element of the queue does not have a next element, add the new element to the tail of the queue
		if tail == LoadItem(&q.tail) {
			if lastNext == nil {
				// 如果尾部元素的下一个元素为空，尝试将新元素添加到尾部元素的下一个位置
				// If the next element of the tail element is null, try to add the new element to the next position of the tail element
				if CasItem(&tail.next, lastNext, newItem) {
					// 如果添加成功，更新队列的尾部元素为新元素
					// If the addition is successful, update the tail element of the queue to the new element
					CasItem(&q.tail, tail, newItem)
					// 使用原子操作增加队列长度
					// Increase the length of the queue using atomic operation
					atomic.AddUint64(&q.length, 1)
					return
				}
			} else {
				// 如果尾部元素的下一个元素不为空，更新队列的尾部元素为尾部元素的下一个元素
				// If the next element of the tail element is not null, update the tail element of the queue to the next element of the tail element
				CasItem(&q.tail, tail, lastNext)
			}
		}
	}
}

// Pop 从队列中移除并返回第一个元素
// Pop removes and returns the first element from the queue.
func (q *LockFreeQueue) Pop() interface{} {
	for {
		// 获取队列头部元素
		// Get the head element of the queue
		head := LoadItem(&q.head)

		// 获取队列尾部元素
		// Get the tail element of the queue
		tail := LoadItem(&q.tail)

		// 获取队列头部元素的下一个元素
		// Get the next element of the head element
		firstNext := LoadItem(&head.next)

		// 如果队列头部元素等于队列尾部元素，就将队列尾部元素的下一个元素设置为队列头部元素
		// If the head element of the queue is equal to the tail element of the queue,
		// set the next element of the tail element of the queue to the head element of the queue
		if head == LoadItem(&q.head) {
			if head == tail {
				// 如果队列为空，就返回nil
				// If the queue is empty, return nil
				if firstNext == nil {
					return nil
				}

				// 将队列尾部元素的下一个元素设置为队列头部元素
				// Set the next element of the tail element of the queue to the head element of the queue
				CasItem(&q.tail, tail, firstNext)
			} else {
				// 获取弹出元素的值
				// Get the value of the popped element
				value := firstNext.value

				// 将队列头部元素的下一个元素设置为队列头部元素
				// Set the next element of the head element of the queue to the head element of the queue
				if CasItem(&q.head, head, firstNext) {
					// 队列长度减1
					// Decrement the length of the queue
					atomic.AddUint64(&q.length, ^uint64(0))

					// 清除引用，帮助GC
					// Clear the reference to help GC
					head.next = nil
					head.value = nil

					// 返回弹出元素的值
					// Return the value of the popped element
					return value
				}
			}
		}
	}
}

// / Len 返回队列中的元素数量
// Len returns the number of elements in the queue.
func (q *LockFreeQueue) Len() uint64 {
	// 使用原子操作获取队列长度
	// Use atomic operation to get the length of the queue
	return atomic.LoadUint64(&q.length)
}

// Reset 将队列重置为初始状态
// Reset resets the queue to its initial state.
func (q *LockFreeQueue) Reset() {
	// 创建一个新的元素
	// Create a new element
	head := Element{}

	// 将队列头指针指向新元素
	// Set the head pointer of the queue to the new element
	q.head = unsafe.Pointer(&head)

	// 将队列尾指针指向新元素
	// Set the tail pointer of the queue to the new element
	q.tail = unsafe.Pointer(&head)

	// 使用原子操作将队列长度置为0
	// Use atomic operation to set the length of the queue to 0
	atomic.StoreUint64(&q.length, 0)
}
