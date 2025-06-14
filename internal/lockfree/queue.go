package lockfree

import (
	"sync/atomic"
	"unsafe"
)

// 缓存行大小（字节）
const cacheLinePadSize = 64

// 填充大小（以uint64为单位）
const paddingSize = (cacheLinePadSize - 8) / 8

// loadNode 函数用于加载指定指针 p 指向的 Node 结构体
func loadNode(p *unsafe.Pointer) *Node {
	// 使用 atomic.LoadPointer 加载并返回指定指针 p 指向的 Node 结构体
	return (*Node)(atomic.LoadPointer(p))
}

// compareAndSwapNode 函数用于比较并交换指定指针 p 指向的 Node 结构体
func compareAndSwapNode(p *unsafe.Pointer, old, new *Node) bool {
	// 使用 atomic.CompareAndSwapPointer 比较并交换指定指针 p 指向的 Node 结构体
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}

// LockFreeQueue 是一个无锁队列结构体
type LockFreeQueue struct {
	// pool 是一个 NodePool 结构体实例的指针
	pool *NodePool

	// padding1 避免 pool 和 length 在同一缓存行
	_padding1 [paddingSize]uint64

	// length 是队列的长度
	length int64

	// padding2 避免 length 和 head 在同一缓存行
	_padding2 [paddingSize]uint64

	// head 是指向队列头部的指针
	head unsafe.Pointer

	// padding3 避免 head 和 tail 在同一缓存行
	_padding3 [paddingSize]uint64

	// tail 是指向队列尾部的指针
	tail unsafe.Pointer

	// padding4 避免 tail 和其他数据在同一缓存行
	_padding4 [paddingSize]uint64
}

// NewLockFreeQueue 函数用于创建一个新的 LockFreeQueue 结构体实例。
func NewLockFreeQueue() *LockFreeQueue {
	// 创建一个新的 Node 结构体实例。
	firstNode := NewNode(nil)

	// 返回一个新的 LockFreeQueue 结构体实例。
	return &LockFreeQueue{
		pool: NewNodePool(),
		head: unsafe.Pointer(firstNode),
		tail: unsafe.Pointer(firstNode),
	}
}

// Push 方法用于将一个值添加到 LockFreeQueue 队列的末尾
func (q *LockFreeQueue) Push(value interface{}) {
	// 从 NodePool 中获取一个新的 Node 实例
	node := q.pool.Get()

	// 将新节点的 value 字段设置为传入的值
	node.value = value

	// 使用无限循环来尝试将新节点添加到队列的末尾
	for {
		// 减少原子操作：一次性缓存队列尾节点，避免多次读取
		tail := loadNode(&q.tail)
		next := loadNode(&tail.next)

		// 检查尾节点是否仍然是队列的尾节点
		if tail == loadNode(&q.tail) {
			// 如果尾节点的下一个节点是 nil，说明尾节点是队列的最后一个节点
			if next == nil {
				// 尝试将尾节点的下一个节点设置为新节点
				if compareAndSwapNode(&tail.next, next, node) {
					// 如果成功，那么将队列的尾节点设置为新节点
					// 注意：即使这个CAS失败也没关系，其他线程会帮助更新
					compareAndSwapNode(&q.tail, tail, node)

					// 并增加队列的长度
					atomic.AddInt64(&q.length, 1)

					// 然后返回，结束函数
					return
				}
			} else {
				// 如果尾节点的下一个节点不是 nil，说明尾节点不是队列的最后一个节点
				// 尝试帮助更新尾指针，减少队列不一致状态的持续时间
				compareAndSwapNode(&q.tail, tail, next)
			}
		}
	}
}

// Pop 方法用于从 LockFreeQueue 队列的头部移除并返回一个值
func (q *LockFreeQueue) Pop() interface{} {
	// 使用无限循环来尝试从队列的头部移除一个值
	for {
		// 减少原子操作：缓存头节点、尾节点和头节点的下一个节点
		head := loadNode(&q.head)
		tail := loadNode(&q.tail)
		first := loadNode(&head.next)

		// 检查头节点是否仍然是队列的头节点
		if head == loadNode(&q.head) {
			// 如果头节点等于尾节点
			if head == tail {
				// 如果头节点的下一个节点是 nil，说明队列是空的，返回 nil
				if first == nil {
					return nil
				}

				// 如果头节点的下一个节点不是 nil，说明尾节点落后了
				// 尝试帮助更新尾指针，减少队列不一致状态的持续时间
				compareAndSwapNode(&q.tail, tail, first)
			} else {
				// 缓存结果值以减少对共享内存的访问
				result := first.value

				// 如果头节点不等于尾节点，尝试将队列的头节点设置为头节点的下一个节点
				if compareAndSwapNode(&q.head, head, first) {
					// 如果成功，那么减少队列的长度
					atomic.AddInt64(&q.length, -1)

					// 重置头节点，将其归还 NodePool
					q.pool.Put(head)

					// 返回缓存的结果值
					return result
				}
			}
		}
	}
}

// Length 方法用于获取 LockFreeQueue 队列的长度
func (q *LockFreeQueue) Length() int64 {
	// 使用 atomic.LoadInt64 函数获取队列的长度
	return atomic.LoadInt64(&q.length)
}

// Reset 方法用于重置 LockFreeQueue 队列
func (q *LockFreeQueue) Reset() {
	// 创建一个新的 Node 结构体实例
	fristNode := NewNode(nil)

	// 将队列的头节点和尾节点都设置为新创建的节点
	q.head = unsafe.Pointer(fristNode)
	q.tail = unsafe.Pointer(fristNode)

	// 重置队列长度
	atomic.StoreInt64(&q.length, 0)
}
