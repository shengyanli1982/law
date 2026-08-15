package law

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/shengyanli1982/law/internal/poller"
	wr "github.com/shengyanli1982/law/internal/writer"
)

// 错误定义
var (
	ErrorWriteAsyncerIsClosed = errors.New("write asyncer is closed")
	ErrorWriteContentIsNil    = errors.New("write content is nil")
)

// pushChecker 是 writer 内部使用的私有接口，用于检测队列是否有可用空间。
// 仅当底层队列实现了 Available() 方法（如 MPSCQueue）时才会触发背压通知。
type pushChecker interface {
	Available() bool
}

type boundedChecker interface {
	IsBounded() bool
}

// WriteAsyncer 异步写入器结构体
type WriteAsyncer struct {
	config         *Config
	queue          Queue
	writer         io.Writer
	bufferedWriter *bufio.Writer
	poller         *poller.Poller
	once           sync.Once
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	state          *wr.Status
	bufferpool     *wr.BufferPool
	hasPushChecker bool
	pushChecker    pushChecker
	// aborted 标记实例已被 StopWithTimeout 超时放弃，
	// 防止后续 Stop 等待被泄漏的关闭 goroutine 占用的 once.Do 而永久阻塞。
	aborted atomic.Bool
}

// NewWriteAsyncer 创建新的异步写入器
func NewWriteAsyncer(writer io.Writer, conf *Config) *WriteAsyncer {
	if writer == nil {
		writer = os.Stdout
	}

	conf = isConfigValid(conf)
	queue := conf.queue

	wa := &WriteAsyncer{
		config:         conf,
		queue:          queue,
		writer:         writer,
		bufferedWriter: bufio.NewWriterSize(writer, conf.buffSize),
		state:          wr.NewStatus(),
		once:           sync.Once{},
		wg:             sync.WaitGroup{},
		bufferpool:     wr.NewBufferPool(),
	}

	wa.ctx, wa.cancel = context.WithCancel(context.Background())
	wa.state.SetRunning(true)

	if pc, ok := wa.queue.(pushChecker); ok {
		if bc, ok2 := wa.queue.(boundedChecker); ok2 && bc.IsBounded() {
			wa.hasPushChecker = true
			wa.pushChecker = pc
		}
	}

	wa.poller = poller.NewPoller(&poller.Config{
		Queue:             queue,
		Writer:            wa.bufferedWriter,
		Callback:          conf.callback,
		BufferPool:        wa.bufferpool,
		HeartbeatInterval: conf.heartbeatInterval,
		IdleTimeout:       conf.idleTimeout,
	})

	wa.wg.Add(1)
	go wa.poller.Run(wa.ctx, &wa.wg)

	return wa
}

// Stop 停止异步写入器。
// 若此前的 StopWithTimeout 已超时，实例被永久放弃，本方法立即返回，不执行任何关闭动作。
// 被放弃的实例不可通过重试恢复，调用方应直接退出进程。
// 注意：Stop 与 StopWithTimeout 不得并发调用（并发调用且底层 I/O 永久卡死时，
// 未感知 aborted 标记的调用方可能永久阻塞在关闭流程上）。
func (wa *WriteAsyncer) Stop() {
	if wa.aborted.Load() {
		return
	}

	wa.once.Do(func() {
		wa.state.SetRunning(false)
		// 关闭队列以唤醒阻塞在有界队列 Push() 上的 goroutine
		if closer, ok := wa.queue.(io.Closer); ok {
			closer.Close()
		}
		wa.cancel()
		wa.wg.Wait()
		wa.poller.CleanQueue()
		if err := wa.bufferedWriter.Flush(); err != nil {
			if wa.config.callback != nil {
				wa.config.callback.OnWriteFailed(nil, err)
			}
		}
		wa.bufferedWriter.Reset(io.Discard)
	})
}

// StopWithTimeout 带超时的停止方法，防止底层 I/O 卡住时无限阻塞。
// 超时返回 context.DeadlineExceeded；正常关闭返回 nil。
// 超时即表示该实例被永久放弃：后台关闭工作可能仍在进行且不会被回收，
// 实例不可通过重试恢复，调用方应直接退出进程。
// 注意：Stop 与 StopWithTimeout 不得并发调用（并发调用且底层 I/O 永久卡死时，
// 未感知 aborted 标记的调用方可能永久阻塞在关闭流程上）。
func (wa *WriteAsyncer) StopWithTimeout(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		wa.Stop()
		close(done)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-timer.C:
		wa.state.SetRunning(false)
		wa.aborted.Store(true)
		return context.DeadlineExceeded
	}
}

// Write 实现写入方法
func (wa *WriteAsyncer) Write(p []byte) (n int, err error) {
	if !wa.state.IsRunning() {
		return 0, ErrorWriteAsyncerIsClosed
	}

	if p == nil {
		return 0, ErrorWriteContentIsNil
	}

	l := len(p)
	if l <= 0 {
		return 0, nil
	}

	buff := wa.bufferpool.GetWithHint(l)
	if buff.Cap() < l {
		buff.Grow(l - buff.Cap())
	}

	if n, err = buff.Write(p); err != nil {
		wa.bufferpool.Put(buff)
		return 0, err
	}

	if wa.hasPushChecker {
		if !wa.pushChecker.Available() {
			wa.config.callback.OnWriteBlocked("bounded queue full, push will block")
		}
	}
	wa.queue.Push(buff)
	return l, nil
}

// WriteString 实现 io.StringWriter 接口，使日志框架（zap/logrus/stdlib log）
// 检测到该接口时自动走字符串写入路径，避免 string→[]byte 的额外分配。
func (wa *WriteAsyncer) WriteString(s string) (n int, err error) {
	if !wa.state.IsRunning() {
		return 0, ErrorWriteAsyncerIsClosed
	}

	if s == "" {
		return 0, nil
	}

	l := len(s)
	buff := wa.bufferpool.GetWithHint(l)
	if buff.Cap() < l {
		buff.Grow(l - buff.Cap())
	}

	src := unsafe.Slice(unsafe.StringData(s), l)
	if n, err = buff.Write(src); err != nil {
		wa.bufferpool.Put(buff)
		return 0, err
	}

	if wa.hasPushChecker {
		if !wa.pushChecker.Available() {
			wa.config.callback.OnWriteBlocked("bounded queue full, push will block")
		}
	}
	wa.queue.Push(buff)
	return l, nil
}
