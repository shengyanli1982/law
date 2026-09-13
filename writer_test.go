package law

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
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
			_, _ = w.Write(data)
		}
	})

	b.Run("large writes", func(b *testing.B) {
		w := NewWriteAsyncer(&recyclingSink{}, nil)
		defer w.Stop()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = w.Write(largeBytes)
		}
	})

	b.Run("concurrent small writes", func(b *testing.B) {
		w := NewWriteAsyncer(&recyclingSink{}, nil)
		defer w.Stop()

		data := []byte("small")
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = w.Write(data)
			}
		})
	})

	b.Run("string writes", func(b *testing.B) {
		w := NewWriteAsyncer(&recyclingSink{}, nil)
		defer w.Stop()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = w.WriteString("string-small")
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
					_, _ = w.Write(smallData)
				} else {
					_, _ = w.Write(largeBytes)
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
		_, _ = w.Write([]byte("hello"))
	}

	// 在另一个 goroutine 中继续写入（会阻塞在有界队列）
	go func() {
		_, _ = w.Write([]byte("blocked"))
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
	// 即使中途断言失败也解除底层阻塞，让遗留的关闭 goroutine 自行退出，避免 goroutine 泄漏
	defer close(underlying.blockCh)
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

// gateSink 阻塞第一次底层写入直至 gate 关闭，用于卡死 poller 模拟底层 I/O 挂死。
type gateSink struct {
	gate      chan struct{}
	blockOnce sync.Once
	mu        sync.Mutex
	received  []byte
}

func (s *gateSink) Write(p []byte) (int, error) {
	s.blockOnce.Do(func() { <-s.gate })
	s.mu.Lock()
	s.received = append(s.received, p...)
	s.mu.Unlock()
	return len(p), nil
}

// TestWriteAsyncer_Stop_UnblocksBlockedProducer 验证 Stop() 真实关闭有界队列（R3-P0-1）：
// poller 卡死在底层 I/O、队列已满时，阻塞在 Push 上的生产者必须被队列 Close
// 唤醒并在超时内返回，而非永久 park（goroutine 泄漏 + Write 永不返回）。
func TestWriteAsyncer_Stop_UnblocksBlockedProducer(t *testing.T) {
	sink := &gateSink{gate: make(chan struct{})}
	w := NewWriteAsyncer(sink, NewConfig().
		WithQueue(NewBoundedQueue(1, 0)).
		WithBufferSize(1))

	// A 被 bufio 缓冲吸收；B 出队时触发预 flush A，poller 卡死在 gate 上
	_, err := w.Write([]byte("A"))
	require.Nil(t, err)
	time.Sleep(20 * time.Millisecond)
	_, err = w.Write([]byte("B"))
	require.Nil(t, err)
	time.Sleep(100 * time.Millisecond) // poller 现已阻塞在 sink.Write(A)

	// C 占用队列唯一槽位
	_, err = w.Write([]byte("C"))
	require.Nil(t, err)
	time.Sleep(20 * time.Millisecond)

	// D 将阻塞在 Push（队列满且 poller 卡死）
	type writeResult struct {
		n   int
		err error
	}
	resCh := make(chan writeResult, 1)
	go func() {
		n, err := w.Write([]byte("D"))
		resCh <- writeResult{n, err}
	}()
	time.Sleep(200 * time.Millisecond) // 确保 D 已 park 在 notFull.Wait

	// Stop 应关闭队列唤醒 D；Stop 自身会卡在 wg.Wait（poller 被 gate 卡死），放入 goroutine。
	// 用 channel+select 超时判定，不 join 可能泄漏的 goroutine，避免挂死测试套件。
	stopDone := make(chan struct{})
	go func() {
		w.Stop()
		close(stopDone)
	}()

	select {
	case <-resCh:
		// 生产者被队列 Close 唤醒，Push 返回
	case <-time.After(2 * time.Second):
		close(sink.gate) // 放行后台 goroutine，避免污染后续测试
		t.Fatal("blocked producer was not unblocked by Stop() within 2s: queue was never closed")
	}

	close(sink.gate) // 放行 poller，让 Stop 走完剩余关闭流程
	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not complete within 5s after gate release")
	}
}

// errTransientIO 模拟瞬态 I/O 故障（ENOSPC、网络抖动等）。
var errTransientIO = errors.New("transient I/O failure")

// flakySink 可编程失败的 sink：fail 为 true 时写入失败，false 时正常写入，
// 用于模拟底层瞬态故障及恢复。
type flakySink struct {
	mu       sync.Mutex
	fail     bool
	received []byte
}

func (s *flakySink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return 0, errTransientIO
	}
	s.received = append(s.received, p...)
	return len(p), nil
}

func (s *flakySink) setFail(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fail = v
}

func (s *flakySink) got() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return string(s.received)
}

// TestWriteAsyncer_TransientFailure_Recovers 验证瞬态失败后实例可恢复（R3-P0-2）：
// bufio.Writer 的错误是粘性的，底层一次失败后若不 Reset 清除，
// 此后所有写入/flush 都短路返回缓存错误，日志永久全丢并伴随回调风暴。
// 底层恢复后，新写入的数据必须真实到达 sink。
func TestWriteAsyncer_TransientFailure_Recovers(t *testing.T) {
	sink := &flakySink{fail: true}
	cb := &mockCallback{failedCh: make(chan []byte, 64)}
	conf := NewConfig().
		WithCallback(cb).
		WithHeartbeatInterval(20 * time.Millisecond).
		WithIdleTimeout(50 * time.Millisecond)
	w := NewWriteAsyncer(sink, conf)
	defer w.Stop()

	// 第一条：心跳 flush 撞底层失败，触发 OnWriteFailed，bufio 进入粘性错误态
	_, err := w.Write([]byte("first"))
	require.Nil(t, err)

	select {
	case <-cb.failedCh:
		// 失败回调已触发
	case <-time.After(2 * time.Second):
		t.Fatal("OnWriteFailed was not triggered for the transient failure within 2s")
	}

	// 底层恢复
	sink.setFail(false)

	// 恢复后的写入必须真实到达 sink；粘性错误态下将永远丢失
	_, err = w.Write([]byte("second"))
	require.Nil(t, err)

	require.Eventually(t, func() bool {
		return strings.Contains(sink.got(), "second")
	}, 2*time.Second, 10*time.Millisecond,
		"data written after underlying recovery never reached the sink (sticky bufio error)")
}

// TestWriteAsyncer_TransientFailure_Recovers_LargeWrite 验证逐条直写路径的
// 粘性错误恢复（R3-P0-2 路径 c）：载荷不小于 bufio 缓冲时直写底层，
// 失败后必须清除粘性错误，底层恢复后的写入必须真实到达 sink。
func TestWriteAsyncer_TransientFailure_Recovers_LargeWrite(t *testing.T) {
	sink := &flakySink{fail: true}
	cb := &mockCallback{failedCh: make(chan []byte, 64)}
	conf := NewConfig().
		WithCallback(cb).
		WithBufferSize(8).
		WithHeartbeatInterval(20 * time.Millisecond).
		WithIdleTimeout(50 * time.Millisecond)
	w := NewWriteAsyncer(sink, conf)
	defer w.Stop()

	// 16 字节载荷 > 8 字节 bufio 缓冲：直写底层，失败并进入粘性错误态
	_, err := w.Write([]byte("0123456789abcdef"))
	require.Nil(t, err)

	select {
	case <-cb.failedCh:
	case <-time.After(2 * time.Second):
		t.Fatal("OnWriteFailed was not triggered for the direct-write failure within 2s")
	}

	sink.setFail(false)

	_, err = w.Write([]byte("recovered-direct"))
	require.Nil(t, err)

	require.Eventually(t, func() bool {
		return strings.Contains(sink.got(), "recovered-direct")
	}, 2*time.Second, 10*time.Millisecond,
		"data written after recovery never reached the sink (sticky error via direct Write path)")
}

// TestWriteAsyncer_TransientFailure_Recovers_PreFlush 验证预 flush 路径的
// 粘性错误恢复（R3-P0-2 路径 b）：缓冲满触发的先行 Flush 失败后
// 必须清除粘性错误，底层恢复后的写入必须真实到达 sink。
func TestWriteAsyncer_TransientFailure_Recovers_PreFlush(t *testing.T) {
	sink := &flakySink{} // 起始健康
	cb := &mockCallback{failedCh: make(chan []byte, 64)}
	conf := NewConfig().
		WithCallback(cb).
		WithBufferSize(16).
		WithHeartbeatInterval(20 * time.Millisecond).
		WithIdleTimeout(5 * time.Second) // 排除心跳 flush 干扰，确保走预 flush 路径
	w := NewWriteAsyncer(sink, conf)
	defer w.Stop()

	// 第一条 10 字节进入 bufio 缓冲（底层健康，10 < 16 不落盘）
	_, err := w.Write([]byte("0123456789"))
	require.Nil(t, err)
	time.Sleep(100 * time.Millisecond) // 等 poller 拾取并缓冲

	// 底层转为失败；第二条 10 字节 > Available(6) 且 Buffered(10) > 0 → 预 flush 失败
	sink.setFail(true)
	_, err = w.Write([]byte("abcdefghij"))
	require.Nil(t, err)

	select {
	case <-cb.failedCh:
	case <-time.After(2 * time.Second):
		t.Fatal("OnWriteFailed was not triggered for the pre-flush failure within 2s")
	}

	sink.setFail(false)

	// 19 字节 > 16 字节缓冲：直写落盘，无需等待心跳/Stop
	_, err = w.Write([]byte("recovered-pre-flush"))
	require.Nil(t, err)

	require.Eventually(t, func() bool {
		return strings.Contains(sink.got(), "recovered-pre-flush")
	}, 2*time.Second, 10*time.Millisecond,
		"data written after recovery never reached the sink (sticky error via pre-flush path)")
}

// TestWriteAsyncer_WriteDuringStop_ReturnsClosedError 验证 Write×Stop 竞态窗口
// 不得假报成功（R3-P1-1）：生产者阻塞在 Push 上被 Stop 的队列 Close 唤醒后
// 数据已被丢弃，Write 必须返回 ErrorWriteAsyncerIsClosed，
// 而非 (l, nil) 让调用方误判数据已被接受。
func TestWriteAsyncer_WriteDuringStop_ReturnsClosedError(t *testing.T) {
	sink := &gateSink{gate: make(chan struct{})}
	w := NewWriteAsyncer(sink, NewConfig().
		WithQueue(NewBoundedQueue(1, 0)).
		WithBufferSize(1))

	// 布局同 TestWriteAsyncer_Stop_UnblocksBlockedProducer：
	// poller 卡死在 gate 上，C 占用队列唯一槽位，D 阻塞在 Push
	_, err := w.Write([]byte("A"))
	require.Nil(t, err)
	time.Sleep(20 * time.Millisecond)
	_, err = w.Write([]byte("B"))
	require.Nil(t, err)
	time.Sleep(100 * time.Millisecond)
	_, err = w.Write([]byte("C"))
	require.Nil(t, err)
	time.Sleep(20 * time.Millisecond)

	type writeResult struct {
		n   int
		err error
	}
	resCh := make(chan writeResult, 1)
	go func() {
		n, err := w.Write([]byte("D"))
		resCh <- writeResult{n, err}
	}()
	time.Sleep(200 * time.Millisecond) // 确保 D 已 park 在 notFull.Wait

	stopDone := make(chan struct{})
	go func() {
		w.Stop()
		close(stopDone)
	}()

	// D 被队列 Close 唤醒、数据被丢弃：必须报关闭错误，不得假报成功
	select {
	case r := <-resCh:
		require.ErrorIs(t, r.err, ErrorWriteAsyncerIsClosed,
			"Write unblocked by queue Close must report ErrorWriteAsyncerIsClosed, not success")
		require.Equal(t, 0, r.n)
	case <-time.After(2 * time.Second):
		close(sink.gate)
		t.Fatal("blocked producer was not unblocked by Stop() within 2s")
	}

	close(sink.gate) // 放行 poller，让 Stop 走完剩余关闭流程
	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not complete within 5s after gate release")
	}
}

// TestWriteAsyncer_ConcurrentWriteStop_Stress 验证 -race 下并发 Write×Stop
// 不 panic、无数据竞争，错误码空间封闭（只允许 nil 或 ErrorWriteAsyncerIsClosed），
// 且 Stop 完成后 Write 必返回 ErrorWriteAsyncerIsClosed。
func TestWriteAsyncer_ConcurrentWriteStop_Stress(t *testing.T) {
	for i := 0; i < 20; i++ {
		w := NewWriteAsyncer(&recyclingSink{}, NewConfig().WithQueue(NewBoundedQueue(4, 0)))

		var wg sync.WaitGroup
		for g := 0; g < 4; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					if _, err := w.Write([]byte("x")); err != nil {
						assert.ErrorIs(t, err, ErrorWriteAsyncerIsClosed)
					}
				}
			}()
		}
		w.Stop()
		wg.Wait()

		_, err := w.Write([]byte("after-stop"))
		require.ErrorIs(t, err, ErrorWriteAsyncerIsClosed)
	}
}

// TestWriteAsyncer_StopWithTimeout_AbortedRetry 验证已放弃实例上重试
// StopWithTimeout 不得假报干净关闭（R3-P2-5）：首次超时后实例被永久放弃，
// 内部 Stop() 因 aborted 短路立即返回，重试若据此返回 nil 会让调用方
// 误判日志已完整落盘；重试必须仍返回 DeadlineExceeded。
func TestWriteAsyncer_StopWithTimeout_AbortedRetry(t *testing.T) {
	underlying := &blockingWriter{blockCh: make(chan struct{})}
	// 即使中途断言失败也解除底层阻塞，让遗留的关闭 goroutine 自行退出，避免 goroutine 泄漏
	defer close(underlying.blockCh)
	// WithBufferSize(1) 令载荷直达底层 writer，卡住 poller
	w := NewWriteAsyncer(underlying, NewConfig().WithBufferSize(1))

	_, err := w.Write([]byte("0123456789"))
	require.Nil(t, err)

	require.ErrorIs(t, w.StopWithTimeout(50*time.Millisecond), context.DeadlineExceeded)

	// 已放弃实例：重试不得假报干净关闭
	require.ErrorIs(t, w.StopWithTimeout(50*time.Millisecond), context.DeadlineExceeded)
}

// ioCloserQueue 测试用自定义队列：实现 law.Queue 与惯用的 Close() error（io.Closer）形态，
// 并以 closed 通道记录 Close 是否被调用。
// 用于鉴别 Stop() 的 close-on-stop 断言对 io.Closer 形态的覆盖（NEW-P1-1）。
type ioCloserQueue struct {
	mu        sync.Mutex
	items     []*bytes.Buffer
	closed    chan struct{}
	closeOnce sync.Once
}

func (q *ioCloserQueue) Push(value *bytes.Buffer) {
	q.mu.Lock()
	defer q.mu.Unlock()
	select {
	case <-q.closed:
		return // 关闭后丢弃，与内置 MPSCQueue 的关闭语义一致
	default:
	}
	q.items = append(q.items, value)
}

// Pop 遵守 Queue 契约的非阻塞语义：队列为空时立即返回 nil。
func (q *ioCloserQueue) Pop() *bytes.Buffer {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	v := q.items[0]
	q.items = q.items[1:]
	return v
}

// Close 实现 io.Closer 形态（Close() error），幂等。
func (q *ioCloserQueue) Close() error {
	q.closeOnce.Do(func() { close(q.closed) })
	return nil
}

func (q *ioCloserQueue) isClosed() bool {
	select {
	case <-q.closed:
		return true
	default:
		return false
	}
}

// TestWriteAsyncer_Stop_ClosesIoCloserQueue 验证 Stop() 对 Close() error（io.Closer）
// 形态的自定义队列同样执行关闭（NEW-P1-1）：Go 接口方法集精确匹配，
// `interface{ Close() }` 单形态断言对 io.Closer 形态静默 miss，
// 自定义有界队列将失去 Close→唤醒阻塞生产者的路径，graceful shutdown 时 goroutine 泄漏。
func TestWriteAsyncer_Stop_ClosesIoCloserQueue(t *testing.T) {
	q := &ioCloserQueue{closed: make(chan struct{})}
	w := NewWriteAsyncer(io.Discard, NewConfig().WithQueue(q))

	_, err := w.Write([]byte("data"))
	require.Nil(t, err)

	w.Stop()

	require.True(t, q.isClosed(),
		"Stop() must close a queue implementing io.Closer (Close() error), not only interface{ Close() }")
}
