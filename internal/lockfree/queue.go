package lockfree

import (
	"sync/atomic"
	"unsafe"
)

// loadNode 函数用于加载指定指针 p 指向的 Node 结构体
// The loadNode function is used to load the Node struct pointed to by the specified pointer p
func loadNode(p *unsafe.Pointer) *Node {
	// 使用 atomic.LoadPointer 加载并返回指定指针 p 指向的 Node 结构体
	// Uses atomic.LoadPointer to load and return the Node struct pointed to by the specified pointer p
	return (*Node)(atomic.LoadPointer(p))
}

// compareAndSwapNode 函数用于比较并交换指定指针 p 指向的 Node 结构体
// The compareAndSwapNode function is used to compare and swap the Node struct pointed to by the specified pointer p
func compareAndSwapNode(p *unsafe.Pointer, old, new *Node) bool {
	// 使用 atomic.CompareAndSwapPointer 比较并交换指定指针 p 指向的 Node 结构体
	// Uses atomic.CompareAndSwapPointer to compare and swap the Node struct pointed to by the specified pointer p
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}

// LockFreeQueue 是一个无锁队列结构体
// LockFreeQueue is a lock-free queue struct
type LockFreeQueue struct {
	// pool 是一个 NodePool 结构体实例的指针
	// pool is a pointer to an instance of the NodePool struct
	pool *NodePool

	// length 是队列的长度
	// length is the length of the queue
	length int64

	// head 是指向队列头部的指针
	// head is a pointer to the head of the queue
	head unsafe.Pointer

	// tail 是指向队列尾部的指针
	// tail is a pointer to the tail of the queue
	tail unsafe.Pointer
}

// NewLockFreeQueue 函数用于创建一个新的 LockFreeQueue 结构体实例。
// The NewLockFreeQueue function is used to create a new instance of the LockFreeQueue struct.
func NewLockFreeQueue() *LockFreeQueue {
	// 创建一个新的 Node 结构体实例。
	// Create a new Node struct instance.
	firstNode := NewNode(nil)

	// 返回一个新的 LockFreeQueue 结构体实例。
	// Returns a new instance of the LockFreeQueue struct.
	return &LockFreeQueue{
		// 创建一个新的 NodePool 实例，用于管理 Node 实例。
		// Create a new NodePool instance for managing Node instances.
		pool: NewNodePool(),

		// 初始化 head 指针，指向我们刚刚创建的 Node 实例。
		// Initialize the head pointer to point to the Node instance we just created.
		head: unsafe.Pointer(firstNode),

		// 初始化 tail 指针，也指向我们刚刚创建的 Node 实例。
		// Initialize the tail pointer to also point to the Node instance we just created.
		tail: unsafe.Pointer(firstNode),
	}
}

// Push 方法用于将一个值添加到 LockFreeQueue 队列的末尾
// The Push method is used to add a value to the end of the LockFreeQueue queue
func (q *LockFreeQueue) Push(value interface{}) {
	// 从 NodePool 中获取一个新的 Node 实例
	// Get a new Node instance from the NodePool
	node := q.pool.Get()

	// 将新节点的 value 字段设置为传入的值
	// Set the value field of the new node to the passed in value
	node.value = value

	// 使用无限循环来尝试将新节点添加到队列的末尾
	// Use an infinite loop to try to add the new node to the end of the queue
	for {
		// 加载队列的尾节点
		// Load the tail node of the queue
		tail := loadNode(&q.tail)

		// 加载尾节点的下一个节点
		// Load the next node of the tail node
		next := loadNode(&tail.next)

		// 检查尾节点是否仍然是队列的尾节点
		// Check if the tail node is still the tail node of the queue
		if tail == loadNode(&q.tail) {
			// 如果尾节点的下一个节点是 nil，说明尾节点是队列的最后一个节点
			// If the next node of the tail node is nil, it means that the tail node is the last node of the queue
			if next == nil {
				// 尝试将尾节点的下一个节点设置为新节点
				// Try to set the next node of the tail node to the new node
				if compareAndSwapNode(&tail.next, next, node) {
					// 如果成功，那么将队列的尾节点设置为新节点
					// If successful, then set the tail node of the queue to the new node
					compareAndSwapNode(&q.tail, tail, node)

					// 并增加队列的长度
					// And increase the length of the queue
					atomic.AddInt64(&q.length, 1)

					// 然后返回，结束函数
					// Then return to end the function
					return
				}
			} else {
				// 如果尾节点的下一个节点不是 nil，说明尾节点不是队列的最后一个节点，那么将队列的尾节点设置为尾节点的下一个节点
				// If the next node of the tail node is not nil, it means that the tail node is not the last node of the queue, then set the tail node of the queue to the next node of the tail node
				compareAndSwapNode(&q.tail, tail, next)
			}
		}
	}
}

// Pop 方法用于从 LockFreeQueue 队列的头部移除并返回一个值
// The Pop method is used to remove and return a value from the head of the LockFreeQueue queue
func (q *LockFreeQueue) Pop() interface{} {
	// 使用无限循环来尝试从队列的头部移除一个值
	// Use an infinite loop to try to remove a value from the head of the queue
	for {
		// 加载队列的头节点
		// Load the head node of the queue
		head := loadNode(&q.head)

		// 加载队列的尾节点
		// Load the tail node of the queue
		tail := loadNode(&q.tail)

		// 加载头节点的下一个节点
		// Load the next node of the head node
		first := loadNode(&head.next)

		// 检查头节点是否仍然是队列的头节点
		// Check if the head node is still the head node of the queue
		if head == loadNode(&q.head) {
			// 如果头节点等于尾节点
			// If the head node is equal to the tail node
			if head == tail {
				// 如果头节点的下一个节点是 nil，说明队列是空的，返回 nil
				// If the next node of the head node is nil, it means that the queue is empty, return nil
				if first == nil {
					return nil
				}

				// 如果头节点的下一个节点不是 nil，说明尾节点落后了，尝试将队列的尾节点设置为头节点的下一个节点
				// If the next node of the head node is not nil, it means that the tail node is lagging behind, try to set the tail node of the queue to the next node of the head node
				compareAndSwapNode(&q.tail, tail, first)
			} else {
				// 并返回头节点的值
				// And return the value of the head node
				result := first.value

				// 如果头节点不等于尾节点，尝试将队列的头节点设置为头节点的下一个节点
				// If the head node is not equal to the tail node, try to set the head node of the queue to the next node of the head node
				if compareAndSwapNode(&q.head, head, first) {
					// 如果成功，那么减少队列的长度
					// If successful, then decrease the length of the queue
					atomic.AddInt64(&q.length, -1)

					// 重置头节点，将其归还 NodePool
					// Reset the head node and put it back into the NodePool
					q.pool.Put(head)

					// 如果结果不是空值，返回结果
					// If the result is not an empty value, return the result
					return result
				}

				// 如果设置头节点失败，那么将结果设置为 nil
				// If setting the head node fails, then set the result to nil
				result = nil
			}
		}
	}
}

// Length 方法用于获取 LockFreeQueue 队列的长度
// The Length method is used to get the length of the LockFreeQueue queue
func (q *LockFreeQueue) Length() int64 {
	// 使用 atomic.LoadInt64 函数获取队列的长度
	// Use the atomic.LoadInt64 function to get the length of the queue
	return atomic.LoadInt64(&q.length)
}

// Reset 方法用于重置 LockFreeQueue 队列
// The Reset method is used to reset the LockFreeQueue queue
func (q *LockFreeQueue) Reset() {
	// 创建一个新的 Node 结构体实例
	// Create a new Node struct instance
	fristNode := NewNode(nil)

	// 将队列的头节点和尾节点都设置为新创建的节点
	// Set both the head node and the tail node of the queue to the newly created node
	q.head = unsafe.Pointer(fristNode)
	q.tail = unsafe.Pointer(fristNode)

	// 使用 atomic.StoreInt64 函数将队列的长度设置为 0
	// Use the atomic.StoreInt64 function to set the length of the queue to 0
	atomic.StoreInt64(&q.length, 0)
}
