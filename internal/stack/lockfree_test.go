package stack

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLockFreeStack(t *testing.T) {
	s := NewLockFreeStack()

	// Push elements into the stack
	for i := 0; i < 10; i++ {
		s.Push(i)
	}

	// Pop elements from the stack
	for i := 9; i >= 0; i-- {
		v := s.Pop()
		assert.Equal(t, i, v, "Incorrect value popped from the stack. Expected %d, got %d", i, v)
	}
}

func TestLockFreeStackParallel(t *testing.T) {
	nums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	s := NewLockFreeStack()

	// Push elements into the stack
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Push(i)
		}(i)
	}
	wg.Wait()

	// Pop elements from the stack
	wg = sync.WaitGroup{}
	for i := 9; i >= 0; i-- {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v := s.Pop()
			assert.Contains(t, nums, v, "Incorrect value in the queue. Expected %d, got %d", i, v)
		}(i)
	}
	wg.Wait()
}

func TestLockFreeStackParallelAtSametime(t *testing.T) {
	nums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	s := NewLockFreeStack()

	// Push elements into the stack
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Push(i)
		}(i)
	}

	// Pop elements from the stack
	for i := 9; i >= 0; i-- {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v := s.Pop()
			assert.Contains(t, nums, v, "Incorrect value in the queue. Expected %d, got %d", i, v)
		}(i)
	}
	wg.Wait()
}

func BenchmarkLockFreeStack(b *testing.B) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	s := NewLockFreeStack()
	b.ResetTimer()
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			s.Push(i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			s.Pop()
		}
	}()
	wg.Wait()
}

func BenchmarkLockFreeStackParallel(b *testing.B) {
	s := NewLockFreeStack()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Push(1)
			s.Pop()
		}
	})
}
