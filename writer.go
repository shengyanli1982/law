package law

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"time"

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

// Stop 停止异步写入器
func (wa *WriteAsyncer) Stop() {
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

	if pc, ok := wa.queue.(pushChecker); ok {
		if !pc.Available() {
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

	for i := 0; i < l; i++ {
		if err = buff.WriteByte(s[i]); err != nil {
			wa.bufferpool.Put(buff)
			return i, err
		}
	}

	if pc, ok := wa.queue.(pushChecker); ok {
		if !pc.Available() {
			wa.config.callback.OnWriteBlocked("bounded queue full, push will block")
		}
	}
	wa.queue.Push(buff)
	return l, nil
}
