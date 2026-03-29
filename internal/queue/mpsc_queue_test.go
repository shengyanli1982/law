package queue

import (
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
