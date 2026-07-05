package law

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var largeBytes = []byte("#Wqt2ZcQmVzk4KZJPmPamr3cBLNEf5dex2N2RkqhS3E7G6PWnzFEukskx5Z822mZd7")

type callback struct {
	t *testing.T
}

func (c *callback) OnWriteFailed(b []byte, err error) {
	if b != nil {
		fmt.Printf("## callback.OnWriteFailed(%s, %v)\n", b, err)
		assert.Equal(c.t, b, largeBytes, "Expected bytes")
	}
	assert.ErrorIs(c.t, err, errorWriteFailed, "Expected error")
}

func (c *callback) OnWriteBlocked(reason string) {
	fmt.Printf("## callback.OnWriteBlocked(%s)\n", reason)
}

var errorWriteFailed = errors.New("write context failed")

type faultyWriter struct{}

func (fw *faultyWriter) Write(p []byte) (n int, err error) {
	fmt.Printf("!! faultyWriter.Write(%s)\n", p)
	return 0, errorWriteFailed
}

func TestWriteAsyncer_Standard(t *testing.T) {
	buff := bytes.NewBuffer(make([]byte, 0, 1024))

	w := NewWriteAsyncer(buff, nil)

	_, err := w.Write([]byte("hello"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("world"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("!!!"))
	assert.Nil(t, err)

	w.Stop()

	assert.Equal(t, "helloworld!!!", buff.String())
}

func TestWriteAsyncer_WaitForIdleSync(t *testing.T) {
	buff := bytes.NewBuffer(make([]byte, 0, 1024))

	w := NewWriteAsyncer(buff, nil)

	_, err := w.Write([]byte("hello"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("world"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("!!!"))
	assert.Nil(t, err)

	w.Stop()

	assert.Equal(t, "helloworld!!!", buff.String())
}

func TestWriteAsyncer_EarlyShutdown(t *testing.T) {
	buff := bytes.NewBuffer(make([]byte, 0, 1024))

	w := NewWriteAsyncer(buff, nil)

	_, err := w.Write([]byte("hello"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("world"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("!!!"))
	assert.Nil(t, err)

	w.Stop()

	assert.Equal(t, "helloworld!!!", buff.String())

	_, err = w.Write([]byte("stop"))

	assert.ErrorIs(t, err, ErrorWriteAsyncerIsClosed, "Expected error")
	assert.Equal(t, "helloworld!!!", buff.String())
}

func TestWriteAsyncer_OnWriteFailed(t *testing.T) {

	t.Run("Message large than bufferSize", func(t *testing.T) {
		conf := NewConfig().WithCallback(&callback{t: t}).WithBufferSize(60)

		w := NewWriteAsyncer(&faultyWriter{}, conf)
		defer w.Stop()

		for i := 0; i < 10; i++ {
			_, err := w.Write(largeBytes)
			assert.Nil(t, err)
		}

		fmt.Println(">>> Error return by bufio Write method")

		time.Sleep(time.Second)
	})

	t.Run("Message less than bufferSize", func(t *testing.T) {
		conf := NewConfig().WithCallback(&callback{t: t}).WithBufferSize(600)

		w := NewWriteAsyncer(&faultyWriter{}, conf)
		defer w.Stop()

		for i := 0; i < 10; i++ {
			_, err := w.Write(largeBytes)
			assert.Nil(t, err)
		}

		fmt.Println(">>> Error return by bufio Flush method")

		time.Sleep(time.Second)
	})
}

func TestWriteAsyncer_EdgeCases(t *testing.T) {
	t.Run("nil writer defaults to stdout", func(t *testing.T) {
		w := NewWriteAsyncer(nil, nil)
		assert.NotNil(t, w)
		w.Stop()
	})

	t.Run("nil content", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0))
		w := NewWriteAsyncer(buff, nil)
		defer w.Stop()

		_, err := w.Write(nil)
		assert.ErrorIs(t, err, ErrorWriteContentIsNil)
	})

	t.Run("empty content", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0))
		w := NewWriteAsyncer(buff, nil)
		defer w.Stop()

		n, err := w.Write([]byte{})
		assert.Nil(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("multiple stop calls", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0))
		w := NewWriteAsyncer(buff, nil)

		w.Stop()
		w.Stop()
	})
}

func TestWriteAsyncer_Concurrent(t *testing.T) {
	t.Run("concurrent writes", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0, 1024))
		w := NewWriteAsyncer(buff, nil)

		var wg sync.WaitGroup
		writers := 10
		iterations := 100

		wg.Add(writers)
		for i := 0; i < writers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					content := []byte(fmt.Sprintf("w%d-%d", id, j))
					_, err := w.Write(content)
					assert.Nil(t, err)
				}
			}(i)
		}
		wg.Wait()
		w.Stop()
		assert.Greater(t, buff.Len(), 0)
	})
}

func BenchmarkWriteAsyncer(b *testing.B) {
	b.Run("small writes", func(b *testing.B) {
		buff := bytes.NewBuffer(make([]byte, 0, b.N*10))
		w := NewWriteAsyncer(buff, nil)
		defer w.Stop()

		data := []byte("small")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w.Write(data)
		}
	})

	b.Run("large writes", func(b *testing.B) {
		buff := bytes.NewBuffer(make([]byte, 0, b.N*len(largeBytes)))
		w := NewWriteAsyncer(buff, nil)
		defer w.Stop()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w.Write(largeBytes)
		}
	})

	b.Run("concurrent small writes", func(b *testing.B) {
		w := NewWriteAsyncer(bytes.NewBuffer(nil), nil)
		defer w.Stop()

		data := []byte("small")
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				w.Write(data)
			}
		})
	})

	b.Run("string writes", func(b *testing.B) {
		buff := bytes.NewBuffer(make([]byte, 0, b.N*len("string-small")))
		w := NewWriteAsyncer(buff, nil)
		defer w.Stop()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w.WriteString("string-small")
		}
	})

	b.Run("concurrent mixed writes", func(b *testing.B) {
		var counter atomic.Uint64
		w := NewWriteAsyncer(bytes.NewBuffer(nil), nil)
		defer w.Stop()

		smallData := []byte("small")
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if counter.Add(1)%2 == 0 {
					w.Write(smallData)
				} else {
					w.Write(largeBytes)
				}
			}
		})
	})
}

type slowWriter struct {
	delay time.Duration
}

func (sw *slowWriter) Write(p []byte) (int, error) {
	time.Sleep(sw.delay)
	return len(p), nil
}

func TestWriteAsyncer_BoundedQueue_StopUnblocks(t *testing.T) {
	// 创建容量为 1 的有界队列
	q := NewBoundedQueue(1, 0)
	conf := NewConfig().WithQueue(q)

	// 使用一个慢 writer 来让队列保持满
	w := NewWriteAsyncer(&slowWriter{delay: 100 * time.Millisecond}, conf)

	// 写入足够多的数据填满队列
	for i := 0; i < 5; i++ {
		w.Write([]byte("hello"))
	}

	// 在另一个 goroutine 中继续写入（会阻塞在有界队列）
	go func() {
		w.Write([]byte("blocked"))
	}()

	// 等一小段时间让 goroutine 阻塞
	time.Sleep(50 * time.Millisecond)

	// Stop 应该解除阻塞并在合理时间内完成
	stopDone := make(chan struct{})
	go func() {
		w.Stop()
		close(stopDone)
	}()

	select {
	case <-stopDone:
		// Stop 成功完成
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not complete within 5 seconds, likely blocked goroutine on bounded queue")
	}
}

func TestWriteAsyncer_BufferHandling(t *testing.T) {
	t.Run("buffer flush on size exceed", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0))
		conf := NewConfig().WithBufferSize(10)
		w := NewWriteAsyncer(buff, conf)

		_, err := w.Write([]byte("small"))
		assert.Nil(t, err)

		_, err = w.Write([]byte("this is a large content"))
		assert.Nil(t, err)

		w.Stop()
		assert.Contains(t, buff.String(), "small")
	})

	t.Run("buffer flush on idle timeout", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0))
		w := NewWriteAsyncer(buff, nil)

		_, err := w.Write([]byte("test"))
		assert.Nil(t, err)

		time.Sleep(DefaultIdleTimeout + time.Second)
		w.Stop()
		assert.Equal(t, "test", buff.String())
	})
}
