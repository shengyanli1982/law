package queue

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLockFreeQueue(t *testing.T) {
	q := NewLockFreeQueue()

	// Test enqueueing elements into the queue
	for i := 0; i < 100000; i++ {
		q.Push(i)
	}

	// Verify the elements in the queue
	for i := 0; i < 100000; i++ {
		v := q.Pop()
		assert.Equal(t, i, v, "Incorrect value in the queue. Expected %d, got %d", i, v)
	}
}

func TestLockFreeQueueParallel(t *testing.T) {
	nums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	q := NewLockFreeQueue()

	// Test enqueueing elements into the queue
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q.Push(i)
		}(i)
	}
	wg.Wait()

	// Verify the elements in the queue
	wg = sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v := q.Pop()
			if v != i {
				assert.Contains(t, nums, v, "Incorrect value in the queue. Expected %d, got %d", i, v)
			}
		}(i)
	}
	wg.Wait()
}

func TestLockFreeQueueParallelAtSametime(t *testing.T) {
	nums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	q := NewLockFreeQueue()

	// Test enqueueing elements into the queue
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q.Push(i)
		}(i)
	}

	// Verify the elements in the queue
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v := q.Pop()
			if v != i {
				assert.Contains(t, nums, v, "Incorrect value in the queue. Expected %d, got %d", i, v)
			}
		}(i)
	}
	wg.Wait()
}

func BenchmarkLockFreeQueue(b *testing.B) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	q := NewLockFreeQueue()
	b.ResetTimer()
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			q.Push(i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			q.Pop()
		}
	}()
	wg.Wait()
}

func BenchmarkLockFreeQueueParallel(b *testing.B) {
	q := NewLockFreeQueue()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Push(1)
			q.Pop()
		}
	})
}
