package queue

import (
	"context"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMPSCQueue_Standard(t *testing.T) {
	q := NewMPSCQueue()

	for i := 0; i < 1000; i++ {
		q.Push(i)
	}

	for i := 0; i < 1000; i++ {
		v := q.Pop()
		if v == nil {
			t.Fatalf("第 %d 次出队得到 nil", i)
		}
		if v.(int) != i {
			t.Fatalf("出队顺序错误，期望=%d，实际=%v", i, v)
		}
	}

	if q.Pop() != nil {
		t.Fatalf("空队列出队应返回 nil")
	}
}

func TestMPSCQueue_WithLimits_BlockingPush(t *testing.T) {
	q := NewMPSCQueueWithLimits(1, 0)
	q.Push(1)

	done := make(chan struct{})
	go func() {
		q.Push(2)
		close(done)
	}()

	select {
	case <-done:
		t.Fatalf("队列满时 Push 不应立即返回")
	case <-time.After(100 * time.Millisecond):
	}

	v := q.Pop()
	if v == nil || v.(int) != 1 {
		t.Fatalf("第一次出队错误，得到=%v", v)
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatalf("释放空间后 Push 未在预期时间内恢复")
	}

	v = q.Pop()
	if v == nil || v.(int) != 2 {
		t.Fatalf("第二次出队错误，得到=%v", v)
	}
}

func TestMPSCQueue_ConcurrentProducersSingleConsumer(t *testing.T) {
	q := NewMPSCQueue()

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
		v := q.Pop()
		if v == nil {
			runtime.Gosched()
			continue
		}
		consumed.Add(1)
	}

	wg.Wait()
	if produced.Load() != consumed.Load() {
		t.Fatalf("生产与消费数量不一致，produced=%d consumed=%d", produced.Load(), consumed.Load())
	}
}

// 5 分钟 soak 测试：默认跳过，设置 LAW_SOAK=1 后执行。
func TestMPSCQueue_Soak5Minutes_NoCrash(t *testing.T) {
	if os.Getenv("LAW_SOAK") != "1" {
		t.Skip("跳过 soak 测试，设置 LAW_SOAK=1 可执行")
	}

	// soak 场景使用有界阻塞队列，避免无界增长导致 OOM。
	q := NewMPSCQueueWithLimits(1<<16, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	producerN := runtime.GOMAXPROCS(0) * 2
	var produced atomic.Int64
	var consumed atomic.Int64

	var producersWG sync.WaitGroup
	producersWG.Add(producerN)
	for i := 0; i < producerN; i++ {
		go func(id int) {
			defer producersWG.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					q.Push(id)
					produced.Add(1)
				}
			}
		}(i)
	}

	producerDone := make(chan struct{})
	go func() {
		producersWG.Wait()
		close(producerDone)
	}()

	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)

		for {
			v := q.Pop()
			if v != nil {
				consumed.Add(1)
				continue
			}

			select {
			case <-producerDone:
				// 生产者全部结束后，排空剩余数据。
				if q.Len() == 0 {
					return
				}
			default:
				runtime.Gosched()
			}
		}
	}()

	<-producerDone

	select {
	case <-consumerDone:
	case <-time.After(30 * time.Second):
		t.Fatalf("soak 结束后消费者排空超时")
	}

	if produced.Load() != consumed.Load() {
		t.Fatalf("soak 结束计数不一致，produced=%d consumed=%d", produced.Load(), consumed.Load())
	}
}

func benchmarkMPSCQueuePushPop(b *testing.B, q *MPSCQueue) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Push(i)
		_ = q.Pop()
	}
}

func BenchmarkMPSCQueue_PushPop(b *testing.B) {
	benchmarkMPSCQueuePushPop(b, NewMPSCQueue())
}

func BenchmarkMPSCQueue_ParallelPushPop(b *testing.B) {
	q := NewMPSCQueue()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Push(1)
			_ = q.Pop()
		}
	})
}
