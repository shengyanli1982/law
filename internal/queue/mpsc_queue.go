package queue

import (
	"bytes"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type queueNode[T any] struct {
	next  *queueNode[T]
	value T
	size  int
}

// MPSCQueue 是基于 mutex/cond 的链表队列，并使用 sync.Pool 复用节点。
// 该实现面向多生产者、单消费者场景，默认不丢数据。
type MPSCQueue[T any] struct {
	mu      sync.Mutex
	notFull *sync.Cond

	head *queueNode[T]
	tail *queueNode[T]

	count int
	bytes int64

	maxItems int
	maxBytes int64

	nodePool  sync.Pool
	closed    atomic.Bool
	closeOnce sync.Once

	// notify 空→非空通知通道，容量 1：突发通知天然合并为一条。
	// 由 Push 在 1ms 节流后发送，消费者通过 NotifyChan 订阅。
	notify chan struct{}
	// lastNotifyMs 最近一次通知的毫秒时间戳，节流判据（atomic 支持锁外更新）。
	lastNotifyMs atomic.Int64
}

// NewMPSCQueue 创建一个无界泛型队列。
func NewMPSCQueue[T any]() *MPSCQueue[T] {
	q := &MPSCQueue[T]{}
	q.notFull = sync.NewCond(&q.mu)
	q.notify = make(chan struct{}, 1)
	q.nodePool.New = func() any {
		return &queueNode[T]{}
	}

	warmUpCount := max(8, runtime.GOMAXPROCS(0)*4)
	for i := 0; i < warmUpCount; i++ {
		q.nodePool.Put(&queueNode[T]{})
	}

	return q
}

// NewMPSCQueueWithLimits 创建带上限的泛型队列。
// 达到上限后 Push 会阻塞等待可用空间。
// maxItems <= 0 表示不限条数，maxBytes <= 0 表示不限字节数。
func NewMPSCQueueWithLimits[T any](maxItems int, maxBytes int64) *MPSCQueue[T] {
	q := NewMPSCQueue[T]()
	q.maxItems = maxItems
	q.maxBytes = maxBytes
	return q
}

func estimateSize[T any](value T) int {
	switch v := any(value).(type) {
	case *bytes.Buffer:
		return v.Len()
	case []byte:
		return len(v)
	case string:
		return len(v)
	case interface{ Len() int }:
		return v.Len()
	default:
		return 1
	}
}

// isFull 判断队列是否已满。
// 空队列豁免：队列为空时直接放行，保证有界队列至少允许一条数据通过（无论其大小），
// 避免向空队列写入单条超过 maxBytes 的数据时 Push 永久阻塞。
func (q *MPSCQueue[T]) isFull(nextSize int) bool {
	if q.count == 0 && q.bytes == 0 {
		return false
	}
	if q.maxItems > 0 && q.count >= q.maxItems {
		return true
	}
	if q.maxBytes > 0 && q.bytes+int64(nextSize) > q.maxBytes {
		return true
	}
	return false
}

// Push 将值入队，Push(nil) 行为未定义。
// 当配置了上限且队列满时，会阻塞等待空间。
// 队列关闭后，Push 立即返回，不再接受新数据。
func (q *MPSCQueue[T]) Push(value T) {
	if q.closed.Load() {
		return
	}

	size := estimateSize(value)

	node := q.nodePool.Get().(*queueNode[T])
	// node 来自 pool 且尚未链入，其他 goroutine 不可见，
	// 赋值移到锁外以收缩临界区（锁内仅保留链表拼接与计数）。
	node.value = value
	node.size = size
	node.next = nil

	q.mu.Lock()
	for q.isFull(size) {
		if q.closed.Load() {
			q.mu.Unlock()
			q.discardNode(node)
			return
		}
		q.notFull.Wait()
	}
	if q.closed.Load() {
		q.mu.Unlock()
		q.discardNode(node)
		return
	}

	// 在 count++ 前捕获空→非空转换，供解锁后的节流通知判断。
	wasEmpty := q.count == 0

	if q.tail == nil {
		q.head = node
		q.tail = node
	} else {
		q.tail.next = node
		q.tail = node
	}

	q.count++
	q.bytes += int64(size)
	q.mu.Unlock()

	if wasEmpty {
		q.notifyThrottled()
	}
}

// discardNode 归还未被链入的节点：清除已预填的 value，
// 保持池内节点零值约定，避免池持有已废弃数据的引用。
func (q *MPSCQueue[T]) discardNode(node *queueNode[T]) {
	var zero T
	node.value = zero
	q.nodePool.Put(node)
}

// notifyThrottled 发送空→非空通知，节流窗口 1ms。
// 前轮无节流实现中，通知开销与"空→非空转换频率"成正比：
// 最快的顺序路径上消费者追平后队列 0↔1 振荡，每条写入都触发
// chan send + 消费者抢锁，导致 string_writes 退化 +15%~+21%。
// 节流后窗口内的多次转换合并为一次通知；竞态良性，
// 最坏多发一次通知，由 chan 容量 1 的 select default 合并。
func (q *MPSCQueue[T]) notifyThrottled() {
	nowMs := time.Now().UnixMilli()
	if nowMs-q.lastNotifyMs.Load() >= 1 {
		q.lastNotifyMs.Store(nowMs)
		select {
		case q.notify <- struct{}{}:
		default:
		}
	}
}

// NotifyChan 返回队列的空→非空通知通道（只读）。
// 语义：通知经 1ms 节流合并，拾取延迟典型 <10ms；
// 极端情况下（单条写入恰好落在节流窗口内且后续无写入），
// 退化为心跳周期拾取。
func (q *MPSCQueue[T]) NotifyChan() <-chan struct{} {
	return q.notify
}

// Close 关闭队列，唤醒所有阻塞在 Push 的 goroutine。
// 关闭后不再接受新数据，Pop 在队列为空时返回零值。
func (q *MPSCQueue[T]) Close() {
	q.closeOnce.Do(func() {
		q.closed.Store(true)
		q.notFull.Broadcast()
	})
}

// Pop 出队一个值；队列为空时返回 T 的零值；
// 队列已关闭且为空时也返回零值。
func (q *MPSCQueue[T]) Pop() T {
	var zero T

	q.mu.Lock()
	node := q.head
	if node == nil {
		q.mu.Unlock()
		return zero
	}

	q.head = node.next
	if q.head == nil {
		q.tail = nil
	}
	q.count--
	q.bytes -= int64(node.size)
	needSignal := q.maxItems > 0 || q.maxBytes > 0
	q.mu.Unlock()
	if needSignal {
		q.notFull.Signal()
	}

	value := node.value
	node.value = zero
	node.next = nil
	node.size = 0
	q.nodePool.Put(node)
	return value
}

// PopBatch 一次性出队至多 max 个元素，按入队顺序返回；
// 队列为空或 max <= 0 时返回 nil。
// 批量语义与实现细节见 PopBatchInto。
func (q *MPSCQueue[T]) PopBatch(max int) []T {
	return q.PopBatchInto(nil, max)
}

// PopBatchInto 一次性出队至多 max 个元素并追加到 dst，返回追加后的切片；
// 队列为空或 max <= 0 时原样返回 dst。
// 相较逐条 Pop 只加锁一次：链表段整体摘除（head/tail/count/bytes 一次更新），
// 节点在锁外逐个归还 nodePool，摊薄消费者侧锁开销。
// dst 容量不足时自动扩容；poller 以持久缓冲调用，避免每批分配新切片。
// 有界队列成功出队后以 Broadcast 唤醒全部阻塞的生产者——
// 批量释放了多个空位，若用 Signal 只唤醒一个生产者，其余将停摆。
func (q *MPSCQueue[T]) PopBatchInto(dst []T, max int) []T {
	if max <= 0 {
		return dst
	}

	q.mu.Lock()
	segment := q.head
	if segment == nil {
		q.mu.Unlock()
		return dst
	}

	// 锁内定位截断点并累计摘除段字节数
	n := 0
	var removedBytes int64
	var cut *queueNode[T] // 摘除段的最后一个节点
	for cur := segment; cur != nil && n < max; cur = cur.next {
		cut = cur
		removedBytes += int64(cur.size)
		n++
	}

	// 整段摘除：rest 为剩余段首，nil 表示队列已摘空
	rest := cut.next
	cut.next = nil
	q.head = rest
	if rest == nil {
		q.tail = nil
	}
	q.count -= n
	q.bytes -= removedBytes
	bounded := q.maxItems > 0 || q.maxBytes > 0
	q.mu.Unlock()

	if bounded {
		q.notFull.Broadcast()
	}

	// 扩容 dst（调用方通常传入 [:0] 的持久缓冲，稳态零分配）
	start := len(dst)
	if need := start + n; need > cap(dst) {
		grown := make([]T, need)
		copy(grown, dst)
		dst = grown
	}
	dst = dst[:start+n]

	// 锁外归还节点并收集值
	var zero T
	i := start
	for node := segment; node != nil; {
		dst[i] = node.value
		i++
		next := node.next
		node.value = zero
		node.next = nil
		node.size = 0
		q.nodePool.Put(node)
		node = next
	}
	return dst
}

// Len 返回当前队列中的元素数量。
func (q *MPSCQueue[T]) Len() int {
	q.mu.Lock()
	n := q.count
	q.mu.Unlock()
	return n
}

// Available 返回队列是否有可用空间接收新元素。
// 无界队列始终返回 true；有界队列在满时返回 false。
func (q *MPSCQueue[T]) Available() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return !q.isFull(1)
}

func (q *MPSCQueue[T]) IsBounded() bool {
	return q.maxItems > 0 || q.maxBytes > 0
}
