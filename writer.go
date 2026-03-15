package law

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/shengyanli1982/law/internal/poller"
	wr "github.com/shengyanli1982/law/internal/writer"
)

// 错误定义
var (
	ErrorWriteAsyncerIsClosed = errors.New("write asyncer is closed")
	ErrorWriteContentIsNil    = errors.New("write content is nil")
)

// WriteAsyncer 异步写入器结构体
type WriteAsyncer struct {
	config         *Config
	queue          Queue
	writer         io.Writer
	bufferedWriter *bufio.Writer
	poller         *poller.Poller
	timer          atomic.Int64
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
		timer:          atomic.Int64{},
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
		Timer:             &wa.timer,
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
		wa.cancel()
		wa.wg.Wait()
		wa.poller.CleanQueue()
		_ = wa.bufferedWriter.Flush()
		wa.bufferedWriter.Reset(io.Discard)
	})
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

	wa.queue.Push(buff)
	return l, nil
}
