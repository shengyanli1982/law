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
		Underlying:        wa.writer,
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
		// 关闭队列以唤醒阻塞在有界队列 Push() 上的 goroutine。
		// 两种 Close 签名在 Go 中互斥（同名方法不可重载），类型开关覆盖双形态：
		// 内置 MPSCQueue 为 Close()（无返回值，config.go 编译期断言防签名漂移）；
		// 自定义队列可能惯用实现 io.Closer（Close() error），单形态断言会对其静默 miss。
		switch closer := wa.queue.(type) {
		case interface{ Close() }:
			closer.Close()
		case io.Closer:
			// Stop 对队列关闭失败无补救动作，与内置 Close() 路径语义一致，忽略错误。
			_ = closer.Close()
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
	// 已放弃的实例不可恢复：内部 Stop() 会因 aborted 短路立即返回，
	// 若放行将使 done 立即关闭、重试假报"干净关闭"，故直接返回超时错误。
	if wa.aborted.Load() {
		return context.DeadlineExceeded
	}

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

// Write 实现写入方法。
// 并发语义：Write 与 Stop 之间存在竞态窗口——IsRunning 检查通过后 Stop 可能
// 已关闭队列，此时数据要么已被 poller 排空落盘、要么被关闭的队列静默丢弃；
// 两种情况 Write 均返回 ErrorWriteAsyncerIsClosed（保守报错，不假报成功）。
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

	// GetWithHint 已按 size 路由到容量足够的池；容量不足时
	// bytes.Buffer.Write 自行扩容（原 Grow(l-Cap) 守卫语义错误，是 no-op，已删除）。
	buff := wa.bufferpool.GetWithHint(l)

	if _, err = buff.Write(p); err != nil {
		wa.bufferpool.Put(buff)
		return 0, err
	}

	if wa.hasPushChecker {
		if !wa.pushChecker.Available() {
			wa.config.callback.OnWriteBlocked("bounded queue full, push will block")
		}
	}
	wa.queue.Push(buff)
	// Push 后复查关闭状态：封堵 IsRunning 检查与 Push 之间的竞态窗口。
	// Stop 进行中时队列已关闭，Push 静默丢弃数据，此处保守报错而非假报成功。
	if !wa.state.IsRunning() {
		return 0, ErrorWriteAsyncerIsClosed
	}
	return l, nil
}

// WriteString 实现 io.StringWriter 接口，使日志框架（zap/logrus/stdlib log）
// 检测到该接口时自动走字符串写入路径，避免 string→[]byte 的额外分配。
// 并发语义同 Write：与 Stop 存在竞态窗口，Push 后复查关闭状态，保守报错不假报成功。
func (wa *WriteAsyncer) WriteString(s string) (n int, err error) {
	if !wa.state.IsRunning() {
		return 0, ErrorWriteAsyncerIsClosed
	}

	if s == "" {
		return 0, nil
	}

	l := len(s)
	// 同 Write：容量由 GetWithHint 路由与 bytes.Buffer.Write 自扩容保证。
	buff := wa.bufferpool.GetWithHint(l)

	src := unsafe.Slice(unsafe.StringData(s), l)
	if _, err = buff.Write(src); err != nil {
		wa.bufferpool.Put(buff)
		return 0, err
	}

	if wa.hasPushChecker {
		if !wa.pushChecker.Available() {
			wa.config.callback.OnWriteBlocked("bounded queue full, push will block")
		}
	}
	wa.queue.Push(buff)
	// Push 后复查关闭状态，语义同 Write：Stop 进行中保守报错，不假报成功。
	if !wa.state.IsRunning() {
		return 0, ErrorWriteAsyncerIsClosed
	}
	return l, nil
}
