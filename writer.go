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
	config         *Config            // 配置信息
	writer         io.Writer          // 底层写入器
	bufferedWriter *bufio.Writer      // 带缓冲的写入器
	poller         *poller.Poller     // 轮询器组件
	timer          atomic.Int64       // 计时器
	once           sync.Once          // 确保只执行一次的控制器
	ctx            context.Context    // 上下文
	cancel         context.CancelFunc // 取消函数
	wg             sync.WaitGroup     // 等待组
	state          *wr.Status         // 状态管理器
	bufferpool     *wr.BufferPool     // 缓冲池

	// 缓存的大小统计，用于预测分配
	avgSize atomic.Int64 // 平均大小
	maxSize atomic.Int64 // 最大大小
}

// NewWriteAsyncer 创建新的异步写入器
func NewWriteAsyncer(writer io.Writer, conf *Config) *WriteAsyncer {
	if writer == nil {
		writer = os.Stdout
	}

	conf = isConfigValid(conf)

	wa := &WriteAsyncer{
		config:         conf,
		writer:         writer,
		bufferedWriter: bufio.NewWriterSize(writer, conf.buffSize),
		state:          wr.NewStatus(),
		timer:          atomic.Int64{},
		once:           sync.Once{},
		wg:             sync.WaitGroup{},
		bufferpool:     wr.NewBufferPool(),
		avgSize:        atomic.Int64{},
		maxSize:        atomic.Int64{},
	}

	wa.ctx, wa.cancel = context.WithCancel(context.Background())
	wa.state.SetExecuteAt(time.Now().UnixMilli())
	wa.state.SetRunning(true)

	// 创建并启动Poller组件
	wa.poller = poller.NewPoller(&poller.Config{
		Queue:             conf.queue,
		Writer:            wa.bufferedWriter,
		Callback:          conf.callback,
		State:             wa.state,
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
		wa.bufferedWriter.Flush()
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

	// 更新最大大小统计
	currentMax := wa.maxSize.Load()
	if int64(l) > currentMax {
		wa.maxSize.Store(int64(l))
	}

	// 使用指数移动平均法(EMA)计算平均大小
	// 权重: 历史数据0.8, 新数据0.2
	avgSize := wa.avgSize.Load()
	if avgSize == 0 {
		wa.avgSize.Store(int64(l))
	} else {
		// EMA = currentAvg * 0.8 + newValue * 0.2
		newAvg := (avgSize * 8 / 10) + (int64(l) * 2 / 10)
		wa.avgSize.Store(newAvg)
	}

	// 获取大小合适的缓冲区，使用EMA统计预测最佳大小
	sizeHint := int(wa.avgSize.Load())
	if sizeHint == 0 {
		sizeHint = l
	}
	buff := wa.bufferpool.GetWithHint(sizeHint)

	// 只在容量不足时扩容
	if buff.Cap() < l {
		buff.Grow(l - buff.Cap())
	}

	if n, err = buff.Write(p); err != nil {
		wa.bufferpool.Put(buff)
		return 0, err
	}

	wa.config.queue.Push(buff)
	return l, nil
}
