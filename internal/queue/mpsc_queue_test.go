package queue

import (
	"bytes"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMPSCQueue_Standard(t *testing.T) {
	q := NewMPSCQueue[int]()

	for i := 0; i < 1000; i++ {
		q.Push(i)
	}

	for i := 0; i < 1000; i++ {
		v := q.Pop()
		require.Equalf(t, i, v, "dequeue order mismatch at iteration %d", i)
	}

	require.Equal(t, 0, q.Pop(), "empty queue should return zero value on pop")
}

func TestMPSCQueue_WithLimits_BlockingPush(t *testing.T) {
	q := NewMPSCQueueWithLimits[int](1, 0)
	q.Push(1)

	done := make(chan struct{})
	go func() {
		q.Push(2)
		close(done)
	}()

	select {
	case <-done:
		require.FailNow(t, "push should block when queue is full")
	case <-time.After(100 * time.Millisecond):
	}

	v := q.Pop()
	require.Equal(t, 1, v, "first pop value mismatch")

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		require.FailNow(t, "push did not resume in time after space was released")
	}

	v = q.Pop()
	require.Equal(t, 2, v, "second pop value mismatch")
}

// TestMPSCQueue_Bounded_OversizedSingleItem 验证空有界队列允许单条超过 maxBytes 的数据通过，
// Push 不会永久阻塞（修复前该测试会阻塞至超时）。
func TestMPSCQueue_Bounded_OversizedSingleItem(t *testing.T) {
	q := NewMPSCQueueWithLimits[*bytes.Buffer](0, 100)

	done := make(chan struct{})
	go func() {
		q.Push(bytes.NewBuffer(bytes.Repeat([]byte("a"), 200)))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		q.Close() // 解除 Push 阻塞，避免 goroutine 泄漏
		require.FailNow(t, "oversized push into empty bounded queue should not block")
	}

	buf := q.Pop()
	require.NotNil(t, buf, "oversized item should be enqueued")
	require.Equal(t, 200, buf.Len(), "oversized item content mismatch")
	require.Equal(t, 0, q.Len(), "queue should be empty after pop")
}

// TestMPSCQueue_Bounded_OversizedDoesNotDisableLimit 验证空队列豁免仅放行一条数据：
// 超大项在队期间，后续 Push 仍受 maxBytes 约束而阻塞，防止豁免被滥用为永久放行。
func TestMPSCQueue_Bounded_OversizedDoesNotDisableLimit(t *testing.T) {
	q := NewMPSCQueueWithLimits[*bytes.Buffer](0, 100)

	// 空队列豁免放行超大项
	q.Push(bytes.NewBuffer(bytes.Repeat([]byte("a"), 200)))
	require.Equal(t, 1, q.Len(), "oversized item should be enqueued")

	// 队列被超大项占用时，正常 Push 应受 maxBytes 约束而阻塞
	done := make(chan struct{})
	go func() {
		q.Push(bytes.NewBuffer(bytes.Repeat([]byte("b"), 10)))
		close(done)
	}()

	select {
	case <-done:
		require.FailNow(t, "push should block while oversized item occupies the queue")
	case <-time.After(100 * time.Millisecond):
	}

	// 弹出超大项后队列重新为空，被阻塞的 Push 应恢复
	buf := q.Pop()
	require.Equal(t, 200, buf.Len(), "first popped item should be the oversized one")

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		q.Close()
		require.FailNow(t, "push did not resume in time after space was released")
	}

	buf = q.Pop()
	require.NotNil(t, buf, "second item should be enqueued")
	require.Equal(t, 10, buf.Len(), "second popped item mismatch")
	require.Equal(t, 0, q.Len(), "queue should be empty after drain")
}

func TestMPSCQueue_ConcurrentProducersSingleConsumer(t *testing.T) {
	q := NewMPSCQueue[int]()

	const producers = 8
	const perProducer = 20000

	var wg sync.WaitGroup
	wg.Add(producers)

	var produced atomic.Int64
	for p := 0; p < producers; p++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perProducer; i++ {
				q.Push(i)
				produced.Add(1)
			}
		}()
	}

	target := int64(producers * perProducer)
	var consumed atomic.Int64
	for consumed.Load() < target {
		if q.Len() == 0 {
			runtime.Gosched()
			continue
		}
		_ = q.Pop()
		consumed.Add(1)
	}

	wg.Wait()
	producedCount := produced.Load()
	consumedCount := consumed.Load()
	require.Equalf(t, producedCount, consumedCount, "produced and consumed counts mismatch: produced=%d consumed=%d", producedCount, consumedCount)
}

// 5-minute soak test: skipped by default; enable with LAW_SOAK=1.
// func TestMPSCQueue_Soak5Minutes_NoCrash(t *testing.T) {
// 	if os.Getenv("LAW_SOAK") != "1" {
// 		t.Skip("skip soak test; set LAW_SOAK=1 to enable")
// 	}
//
// 	// Use a bounded blocking queue to avoid unbounded growth and OOM.
// 	q := NewMPSCQueueWithLimits[int](1<<16, 0)
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
// 	defer cancel()
//
// 	producerN := runtime.GOMAXPROCS(0) * 2
// 	var produced atomic.Int64
// 	var consumed atomic.Int64
//
// 	var producersWG sync.WaitGroup
// 	producersWG.Add(producerN)
// 	for i := 0; i < producerN; i++ {
// 		go func(id int) {
// 			defer producersWG.Done()
// 			for {
// 				select {
// 				case <-ctx.Done():
// 					return
// 				default:
// 					q.Push(id)
// 					produced.Add(1)
// 				}
// 			}
// 		}(i)
// 	}
//
// 	producerDone := make(chan struct{})
// 	go func() {
// 		producersWG.Wait()
// 		close(producerDone)
// 	}()
//
// 	consumerDone := make(chan struct{})
// 	go func() {
// 		defer close(consumerDone)
//
// 		for {
// 			if q.Len() > 0 {
// 				_ = q.Pop()
// 				consumed.Add(1)
// 				continue
// 			}
//
// 			select {
// 			case <-producerDone:
// 				// Drain remaining items after all producers have exited.
// 				if q.Len() == 0 {
// 					return
// 				}
// 			default:
// 				runtime.Gosched()
// 			}
// 		}
// 	}()
//
// 	<-producerDone
//
// 	select {
// 	case <-consumerDone:
// 	case <-time.After(30 * time.Second):
// 		require.FailNow(t, "consumer drain timed out after soak test ended")
// 	}
//
// 	producedCount := produced.Load()
// 	consumedCount := consumed.Load()
// 	require.Equalf(t, producedCount, consumedCount, "soak count mismatch: produced=%d consumed=%d", producedCount, consumedCount)
// }

func TestMPSCQueue_Close_UnblocksPush(t *testing.T) {
	q := NewMPSCQueueWithLimits[int](1, 0)
	q.Push(1)

	done := make(chan struct{})
	go func() {
		q.Push(2)
		close(done)
	}()

	select {
	case <-done:
		require.FailNow(t, "push should block when queue is full")
	case <-time.After(100 * time.Millisecond):
	}

	q.Close()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		require.FailNow(t, "Close did not unblock Push within 1s")
	}
}

func TestMPSCQueue_Close_RejectsNewPush(t *testing.T) {
	q := NewMPSCQueue[int]()
	q.Close()

	q.Push(42)
	require.Equal(t, 0, q.Len(), "Push after Close should not enqueue")
}

func TestMPSCQueue_Close_PopReturnsZeroWhenEmpty(t *testing.T) {
	q := NewMPSCQueue[int]()
	q.Close()
	require.Equal(t, 0, q.Pop(), "Pop on closed empty queue should return zero value")
}

func TestMPSCQueue_Close_Idempotent(t *testing.T) {
	q := NewMPSCQueue[int]()
	require.NotPanics(t, func() {
		for i := 0; i < 100; i++ {
			q.Close()
		}
	})
}

func TestMPSCQueue_Close_DrainAfterClose(t *testing.T) {
	q := NewMPSCQueue[int]()
	for i := 0; i < 5; i++ {
		q.Push(i)
	}
	q.Close()

	for i := 0; i < 5; i++ {
		require.Equal(t, i, q.Pop(), "drain after Close")
	}
	require.Equal(t, 0, q.Pop(), "empty after drain")
}

// TestMPSCQueue_PopBatch_Boundaries 验证 PopBatch 的批量边界：
// 少于/等于/多于 max 的摘取行为、空队列、非正 max。
func TestMPSCQueue_PopBatch_Boundaries(t *testing.T) {
	t.Run("fewer than max", func(t *testing.T) {
		q := NewMPSCQueue[int]()
		for i := 0; i < 3; i++ {
			q.Push(i)
		}

		require.Equal(t, []int{0, 1, 2}, q.PopBatch(5))
		require.Equal(t, 0, q.Len(), "queue should be empty after batch pop")
		require.Nil(t, q.PopBatch(5), "subsequent batch pop on empty queue should return nil")
	})

	t.Run("equal to max", func(t *testing.T) {
		q := NewMPSCQueue[int]()
		for i := 0; i < 5; i++ {
			q.Push(i)
		}

		require.Equal(t, []int{0, 1, 2, 3, 4}, q.PopBatch(5))
		require.Equal(t, 0, q.Len(), "queue should be empty after batch pop")
	})

	t.Run("more than max", func(t *testing.T) {
		q := NewMPSCQueue[int]()
		for i := 0; i < 10; i++ {
			q.Push(i)
		}

		require.Equal(t, []int{0, 1, 2, 3}, q.PopBatch(4))
		require.Equal(t, 6, q.Len(), "remaining count after first batch")
		require.Equal(t, []int{4, 5, 6, 7}, q.PopBatch(4))
		require.Equal(t, []int{8, 9}, q.PopBatch(4), "final batch shorter than max")
		require.Equal(t, 0, q.Len(), "queue should be empty after full drain")
	})

	t.Run("empty queue returns nil", func(t *testing.T) {
		q := NewMPSCQueue[int]()
		require.Nil(t, q.PopBatch(4))
	})

	t.Run("non-positive max returns nil", func(t *testing.T) {
		q := NewMPSCQueue[int]()
		q.Push(1)

		require.Nil(t, q.PopBatch(0), "max=0 should return nil")
		require.Nil(t, q.PopBatch(-1), "max<0 should return nil")
		require.Equal(t, 1, q.Len(), "queue must be intact after non-positive max")
	})
}

// TestMPSCQueue_PopBatch_AfterClose 验证关闭后 PopBatch 仍可排空存量，
// 排空后返回 nil，且 Close 后 Push 的数据不会入队。
func TestMPSCQueue_PopBatch_AfterClose(t *testing.T) {
	q := NewMPSCQueue[int]()
	for i := 0; i < 5; i++ {
		q.Push(i)
	}
	q.Close()

	require.Equal(t, []int{0, 1, 2}, q.PopBatch(3), "drain after Close")
	require.Equal(t, []int{3, 4}, q.PopBatch(10), "drain remainder after Close")
	require.Nil(t, q.PopBatch(3), "closed empty queue should return nil")

	q.Push(99)
	require.Nil(t, q.PopBatch(3), "push after Close must not enqueue")
}

// TestMPSCQueue_PopBatch_BytesAccounting 验证批量出队正确扣减 bytes 计数：
// 摘除释放的字节配额应立即对 maxBytes 约束可见，被阻塞的 Push 恢复。
func TestMPSCQueue_PopBatch_BytesAccounting(t *testing.T) {
	q := NewMPSCQueueWithLimits[*bytes.Buffer](0, 100)
	q.Push(bytes.NewBuffer(bytes.Repeat([]byte("a"), 40)))
	q.Push(bytes.NewBuffer(bytes.Repeat([]byte("b"), 40))) // 已占 80 / 100 字节

	done := make(chan struct{})
	go func() {
		q.Push(bytes.NewBuffer(bytes.Repeat([]byte("c"), 40))) // 80+40 > 100，阻塞
		close(done)
	}()

	select {
	case <-done:
		require.FailNow(t, "push should block while bytes limit is exceeded")
	case <-time.After(100 * time.Millisecond):
	}

	batch := q.PopBatch(10)
	require.Len(t, batch, 2, "batch should contain both enqueued buffers")

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		q.Close() // 解除阻塞，避免 goroutine 泄漏
		require.FailNow(t, "blocked push did not resume after PopBatch freed bytes")
	}
	require.Equal(t, 1, q.Len(), "only the resumed push should remain")
}

// TestMPSCQueue_PopBatch_BroadcastWakesAllProducers 验证有界队列 PopBatch
// 成功后以 Broadcast（而非 Signal）唤醒全部阻塞的生产者：
// 批量释放了多个空位，若误用 Signal 只唤醒一个生产者，
// 其余生产者将永久阻塞（停摆），本测试会超时失败。
func TestMPSCQueue_PopBatch_BroadcastWakesAllProducers(t *testing.T) {
	const producers = 4
	q := NewMPSCQueueWithLimits[int](producers, 0)

	// 填满队列，使后续生产者全部阻塞
	for i := 0; i < producers; i++ {
		q.Push(i)
	}

	var blocked sync.WaitGroup
	blocked.Add(producers)
	for i := 0; i < producers; i++ {
		go func(v int) {
			defer blocked.Done()
			q.Push(v)
		}(i)
	}

	// settle：等所有生产者落到阻塞态
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, producers, q.Len(), "queue should stay full while producers block")

	// 一次摘除全部元素；Broadcast 必须唤醒全部 4 个阻塞生产者
	require.Equal(t, []int{0, 1, 2, 3}, q.PopBatch(producers))

	unblocked := make(chan struct{})
	go func() {
		blocked.Wait()
		close(unblocked)
	}()

	select {
	case <-unblocked:
	case <-time.After(2 * time.Second):
		q.Close() // 解除阻塞，避免 goroutine 泄漏
		require.FailNow(t, "PopBatch must Broadcast: some blocked producers were not woken")
	}

	// 校验存量与内容守恒（唤醒顺序不确定，以总和断言）
	rest := q.PopBatch(producers)
	require.Len(t, rest, producers)
	sum := 0
	for _, v := range rest {
		sum += v
	}
	require.Equal(t, producers*(producers-1)/2, sum, "woken producers' values missing")
}

// TestMPSCQueue_PopBatch_ConcurrentDrain 多生产者持续写入、单消费者批量排空，
// 校验总数守恒（配合 -race 覆盖批量路径的并发安全）。
func TestMPSCQueue_PopBatch_ConcurrentDrain(t *testing.T) {
	q := NewMPSCQueue[int]()

	const producers = 8
	const perProducer = 5000
	const total = producers * perProducer

	var wg sync.WaitGroup
	wg.Add(producers)
	for p := 0; p < producers; p++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perProducer; i++ {
				q.Push(i)
			}
		}()
	}

	var consumed atomic.Int64
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		for consumed.Load() < total {
			batch := q.PopBatch(64)
			if len(batch) == 0 {
				runtime.Gosched()
				continue
			}
			consumed.Add(int64(len(batch)))
		}
	}()

	wg.Wait()
	select {
	case <-consumerDone:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "batch drain did not finish in time")
	}
	require.Equal(t, int64(total), consumed.Load())
	require.Equal(t, 0, q.Len(), "queue should be empty after drain")
}

// TestMPSCQueue_PopBatchInto_Semantics 验证 PopBatchInto 的追加语义：
// 复用调用方缓冲、空队列原样返回、容量不足自动扩容、与 PopBatch 等价。
func TestMPSCQueue_PopBatchInto_Semantics(t *testing.T) {
	t.Run("append into reusable buffer", func(t *testing.T) {
		q := NewMPSCQueue[int]()
		for i := 0; i < 3; i++ {
			q.Push(i)
		}

		buf := make([]int, 0, 8)
		got := q.PopBatchInto(buf, 8)
		require.Equal(t, []int{0, 1, 2}, got)
		require.Len(t, buf, 0, "caller buffer header must stay empty until returned")

		// 二次调用复用同一缓冲
		q.Push(9)
		got = q.PopBatchInto(buf[:0], 8)
		require.Equal(t, []int{9}, got)
	})

	t.Run("empty queue returns dst unchanged", func(t *testing.T) {
		q := NewMPSCQueue[int]()
		buf := make([]int, 0, 4)
		got := q.PopBatchInto(buf, 4)
		require.Empty(t, got, "empty queue must not append")
		require.Same(t, &buf[:1][0], &got[:1][0], "empty queue must keep the caller backing array")
		require.Nil(t, q.PopBatchInto(nil, 4), "nil dst on empty queue stays nil")
		require.Nil(t, q.PopBatchInto(nil, 0), "non-positive max returns dst as-is")
	})

	t.Run("grow when capacity insufficient", func(t *testing.T) {
		q := NewMPSCQueue[int]()
		for i := 0; i < 10; i++ {
			q.Push(i)
		}

		buf := make([]int, 0, 2)
		got := q.PopBatchInto(buf, 10)
		require.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, got)
		require.Equal(t, 0, q.Len())
	})

	t.Run("equivalence with PopBatch", func(t *testing.T) {
		q1 := NewMPSCQueue[int]()
		q2 := NewMPSCQueue[int]()
		for i := 0; i < 100; i++ {
			q1.Push(i)
			q2.Push(i)
		}

		require.Equal(t, q1.PopBatch(64), q2.PopBatchInto(nil, 64))
		require.Equal(t, q1.PopBatch(64), q2.PopBatchInto(nil, 64), "remainder must match too")
	})
}

// TestMPSCQueue_PopBatch_LargeBacklog 验证大规模积压下的批量出队：
// 节点大量进出 nodePool 后数据完整、计数守恒，随后收发复用链路完好。
func TestMPSCQueue_PopBatch_LargeBacklog(t *testing.T) {
	q := NewMPSCQueue[int]()
	const total = 5000
	for i := 0; i < total; i++ {
		q.Push(i)
	}

	batch := q.PopBatch(total)
	require.Len(t, batch, total)
	require.Equal(t, 0, batch[0], "first value mismatch")
	require.Equal(t, total-1, batch[total-1], "last value mismatch")
	require.Equal(t, 0, q.Len())

	// 积压排空后继续收发，验证节点回收复用链路完好
	for i := 0; i < 100; i++ {
		q.Push(i)
	}
	require.Equal(t, 100, q.Len())
	require.Len(t, q.PopBatch(100), 100)
	require.Equal(t, 0, q.Len())
}

func benchmarkMPSCQueuePushPop(b *testing.B, q *MPSCQueue[int]) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Push(i)
		_ = q.Pop()
	}
}

func BenchmarkMPSCQueue_PushPop(b *testing.B) {
	benchmarkMPSCQueuePushPop(b, NewMPSCQueue[int]())
}

func BenchmarkMPSCQueue_ParallelPushPop(b *testing.B) {
	q := NewMPSCQueue[int]()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Push(1)
			_ = q.Pop()
		}
	})
}
