package lockfree

import (
	"sync/atomic"
	"unsafe"
)

// 缓存行大小（字节）
// Cache line size (bytes)
const cacheLinePadSize = 64

// 填充大小（以uint64为单位）
// Padding size (in terms of uint64)
const paddingSize = (cacheLinePadSize - 8) / 8

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

	// padding1 避免 pool 和 length 在同一缓存行
	// padding1 avoids pool and length being in the same cache line
	_padding1 [paddingSize]uint64

	// length 是队列的长度
	// length is the length of the queue
	length int64

	// padding2 避免 length 和 head 在同一缓存行
	// padding2 avoids length and head being in the same cache line
	_padding2 [paddingSize]uint64

	// head 是指向队列头部的指针
	// head is a pointer to the head of the queue
	head unsafe.Pointer

	// padding3 避免 head 和 tail 在同一缓存行
	// padding3 avoids head and tail being in the same cache line
	_padding3 [paddingSize]uint64

	// tail 是指向队列尾部的指针
	// tail is a pointer to the tail of the queue
	tail unsafe.Pointer

	// padding4 避免 tail 和其他数据在同一缓存行
	// padding4 avoids tail and other data being in the same cache line
	_padding4 [paddingSize]uint64
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
		// 减少原子操作：一次性缓存队列尾节点，避免多次读取
		// Reduce atomic operations: cache the tail node once, avoid multiple reads
		tail := loadNode(&q.tail)
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
					// 注意：即使这个CAS失败也没关系，其他线程会帮助更新
					// If successful, then set the tail node of the queue to the new node
					// Note: It's okay if this CAS fails, other threads will help update
					compareAndSwapNode(&q.tail, tail, node)

					// 并增加队列的长度
					// And increase the length of the queue
					atomic.AddInt64(&q.length, 1)

					// 然后返回，结束函数
					// Then return to end the function
					return
				}
			} else {
				// 如果尾节点的下一个节点不是 nil，说明尾节点不是队列的最后一个节点
				// 尝试帮助更新尾指针，减少队列不一致状态的持续时间
				// If the next node of the tail node is not nil, it means that the tail node is not the last node of the queue
				// Try to help update the tail pointer to reduce the duration of queue inconsistency
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
		// 减少原子操作：缓存头节点、尾节点和头节点的下一个节点
		// Reduce atomic operations: cache the head node, tail node, and next node of the head node
		head := loadNode(&q.head)
		tail := loadNode(&q.tail)
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

				// 如果头节点的下一个节点不是 nil，说明尾节点落后了
				// 尝试帮助更新尾指针，减少队列不一致状态的持续时间
				// If the next node of the head node is not nil, it means that the tail node is lagging behind
				// Try to help update the tail pointer to reduce the duration of queue inconsistency
				compareAndSwapNode(&q.tail, tail, first)
			} else {
				// 缓存结果值以减少对共享内存的访问
				// Cache the result value to reduce access to shared memory
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

					// 返回缓存的结果值
					// Return the cached result value
					return result
				}
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

	// 重置队列长度
	// Reset the length of the queue
	atomic.StoreInt64(&q.length, 0)
}
