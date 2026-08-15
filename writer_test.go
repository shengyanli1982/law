package law

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// recyclingSinkLimit 基准汇入器的容量上限：1MiB。
const recyclingSinkLimit = 1 << 20

// recyclingSink 基准专用汇入器：累计内容超过上限时先 Reset 再写，
// 消除无界 bytes.Buffer 增长的 growSlice 分配伪影
// （r0 profile 中占分配 36.6%），让基准反映生产路径的真实开销。
type recyclingSink struct {
	bytes.Buffer
}

func (s *recyclingSink) Write(p []byte) (int, error) {
	if s.Len()+len(p) > recyclingSinkLimit {
		s.Reset()
	}
	return s.Buffer.Write(p)
}

func BenchmarkWriteAsyncer(b *testing.B) {
	b.Run("small writes", func(b *testing.B) {
		w := NewWriteAsyncer(&recyclingSink{}, nil)
		defer w.Stop()

		data := []byte("small")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w.Write(data)
		}
	})

	b.Run("large writes", func(b *testing.B) {
		w := NewWriteAsyncer(&recyclingSink{}, nil)
		defer w.Stop()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w.Write(largeBytes)
		}
	})

	b.Run("concurrent small writes", func(b *testing.B) {
		w := NewWriteAsyncer(&recyclingSink{}, nil)
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
		w := NewWriteAsyncer(&recyclingSink{}, nil)
		defer w.Stop()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w.WriteString("string-small")
		}
	})

	b.Run("concurrent mixed writes", func(b *testing.B) {
		var counter atomic.Uint64
		w := NewWriteAsyncer(&recyclingSink{}, nil)
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

// pickupLatencyWriter 记录从启动到底层 Write 首次收到数据的耗时。
type pickupLatencyWriter struct {
	start time.Time
	ch    chan time.Duration
}

func (w *pickupLatencyWriter) Write(p []byte) (int, error) {
	select {
	case w.ch <- time.Since(w.start):
	default:
	}
	return len(p), nil
}

// TestPoller_PickupLatency 验证队列有数据入队后消费者被即时唤醒
// （P1-2 以 1ms 节流通知重新引入）。
// 保持默认心跳 500ms：无唤醒机制时拾取延迟受心跳节拍约束，最高约 500ms；
// 节流通知下拾取延迟典型 <10ms，断言 50ms 留足余量。
// WithBufferSize(4) 令 bufio 缓冲小于载荷，数据直达底层 writer，只度量拾取时刻。
func TestPoller_PickupLatency(t *testing.T) {
	underlying := &pickupLatencyWriter{ch: make(chan time.Duration, 1)}
	w := NewWriteAsyncer(underlying, NewConfig().WithBufferSize(4))
	defer w.Stop()

	// 等 poller 进入 select 等待（100ms settle），度量稳态拾取延迟
	// 而非启动期首次 drain，避免假 GREEN。
	time.Sleep(100 * time.Millisecond)

	underlying.start = time.Now()
	_, err := w.Write([]byte("hello"))
	require.Nil(t, err)

	select {
	case lat := <-underlying.ch:
		require.Less(t, lat, 50*time.Millisecond, "poller pickup latency should be far below the heartbeat interval")
	case <-time.After(2 * time.Second):
		t.Fatal("underlying writer did not receive data within 2s")
	}
}

// signalWriter 将每次收到的数据拷贝发送到通道，供事件驱动断言。
type signalWriter struct {
	ch chan []byte
}

func (w *signalWriter) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	copy(cp, p)
	select {
	case w.ch <- cp:
	default:
	}
	return len(p), nil
}

// TestWriteAsyncer_TrickleFlush 验证落盘有界性：数据进入 bufio 缓冲后，
// 距上次落盘超过 idleTimeout 必须触发 flush，杜绝 flush 饥饿（P1-3）。
func TestWriteAsyncer_TrickleFlush(t *testing.T) {
	underlying := &signalWriter{ch: make(chan []byte, 8)}
	conf := NewConfig().
		WithIdleTimeout(100 * time.Millisecond).
		WithHeartbeatInterval(20 * time.Millisecond)
	w := NewWriteAsyncer(underlying, conf)
	defer w.Stop()

	_, err := w.Write([]byte("drip"))
	require.Nil(t, err)

	select {
	case data := <-underlying.ch:
		require.Equal(t, []byte("drip"), data)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("buffered data was not flushed within 500ms (trickle flush starvation)")
	}
}

// blockingWriter 阻塞在 Write 上直至 blockCh 关闭，模拟底层 I/O 卡死。
type blockingWriter struct {
	blockCh chan struct{}
}

func (w *blockingWriter) Write(p []byte) (int, error) {
	<-w.blockCh
	return len(p), nil
}

// TestWriteAsyncer_StopWithTimeout_Timeout 验证超时即放弃语义（P1-4）：
// 超时返回 DeadlineExceeded；此后 Stop() 必须立即返回，Write 快速失败。
func TestWriteAsyncer_StopWithTimeout_Timeout(t *testing.T) {
	underlying := &blockingWriter{blockCh: make(chan struct{})}
	// WithBufferSize(1) 令载荷直达底层 writer，卡住 poller
	w := NewWriteAsyncer(underlying, NewConfig().WithBufferSize(1))

	_, err := w.Write([]byte("0123456789"))
	require.Nil(t, err)

	require.ErrorIs(t, w.StopWithTimeout(50*time.Millisecond), context.DeadlineExceeded)

	stopReturned := make(chan struct{})
	go func() {
		w.Stop()
		close(stopReturned)
	}()

	select {
	case <-stopReturned:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Stop() blocked on an aborted instance; expected immediate return")
	}

	_, err = w.Write([]byte("x"))
	require.ErrorIs(t, err, ErrorWriteAsyncerIsClosed)

	// 解除底层阻塞，让遗留的关闭 goroutine 自行退出，避免 goroutine 泄漏
	close(underlying.blockCh)
}

// mockCallback 用原子计数与通道通知记录回调触发情况。
type mockCallback struct {
	failedCount  atomic.Int64
	blockedCount atomic.Int64
	failedCh     chan []byte
}

func (c *mockCallback) OnWriteFailed(content []byte, reason error) {
	c.failedCount.Add(1)
	if c.failedCh != nil {
		cp := make([]byte, len(content))
		copy(cp, content)
		select {
		case c.failedCh <- cp:
		default:
		}
	}
}

func (c *mockCallback) OnWriteBlocked(reason string) {
	c.blockedCount.Add(1)
}

// TestWriteAsyncer_OnWriteFailed_Triggered 验证逐条写入失败时触发回调且 content 携带该条数据。
func TestWriteAsyncer_OnWriteFailed_Triggered(t *testing.T) {
	cb := &mockCallback{failedCh: make(chan []byte, 8)}
	conf := NewConfig().WithCallback(cb).WithBufferSize(60)

	w := NewWriteAsyncer(&faultyWriter{}, conf)
	defer w.Stop()

	// largeBytes 超过 60B 缓冲，bufio 立即写穿并失败
	_, err := w.Write(largeBytes)
	require.Nil(t, err)

	select {
	case content := <-cb.failedCh:
		require.Equal(t, largeBytes, content, "OnWriteFailed should carry the failed content")
	case <-time.After(2 * time.Second):
		t.Fatal("OnWriteFailed was not triggered within 2s")
	}
	require.GreaterOrEqual(t, cb.failedCount.Load(), int64(1))
}

// TestWriteAsyncer_OnWriteBlocked_Triggered 验证有界队列满、Push 即将阻塞时触发 OnWriteBlocked。
func TestWriteAsyncer_OnWriteBlocked_Triggered(t *testing.T) {
	cb := &mockCallback{}
	conf := NewConfig().
		WithCallback(cb).
		WithQueue(NewBoundedQueue(1, 0)).
		WithBufferSize(1) // 载荷大于 bufio 缓冲，消费者被 slowWriter 拖慢，队列持续处于满状态

	w := NewWriteAsyncer(&slowWriter{delay: 100 * time.Millisecond}, conf)
	defer w.Stop()

	for i := 0; i < 4; i++ {
		_, err := w.Write([]byte("0123456789"))
		require.Nil(t, err)
	}

	require.GreaterOrEqual(t, cb.blockedCount.Load(), int64(1), "OnWriteBlocked should be triggered when bounded queue is full")
}

// TestWriteAsyncer_WriteString_Content 验证 WriteString 顺序完整落盘。
func TestWriteAsyncer_WriteString_Content(t *testing.T) {
	buff := bytes.NewBuffer(make([]byte, 0, 1024))
	w := NewWriteAsyncer(buff, nil)

	n, err := w.WriteString("hello ")
	require.Nil(t, err)
	require.Equal(t, 6, n)

	n, err = w.WriteString("world")
	require.Nil(t, err)
	require.Equal(t, 5, n)

	w.Stop()
	require.Equal(t, "hello world", buff.String())
}
