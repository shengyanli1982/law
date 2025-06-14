package law

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	wr "github.com/shengyanli1982/law/internal/writer"
)

// 错误定义
// Error definitions
var (
	ErrorWriteAsyncerIsClosed = errors.New("write asyncer is closed")
	ErrorWriteContentIsNil    = errors.New("write content is nil")
)

// WriteAsyncer 异步写入器结构体
// WriteAsyncer is an asynchronous writer structure
type WriteAsyncer struct {
	config         *Config            // 配置信息 / Configuration
	writer         io.Writer          // 底层写入器 / Underlying writer
	bufferedWriter *bufio.Writer      // 带缓冲的写入器 / Buffered writer
	timer          atomic.Int64       // 计时器 / Timer
	once           sync.Once          // 确保只执行一次的控制器 / Once controller
	ctx            context.Context    // 上下文 / Context
	cancel         context.CancelFunc // 取消函数 / Cancel function
	wg             sync.WaitGroup     // 等待组 / Wait group
	state          *wr.Status         // 状态管理器 / Status manager
	bufferpool     *wr.BufferPool     // 缓冲池 / Buffer pool

	// 缓存的大小统计，用于预测分配
	// Cached size statistics for prediction allocation
	avgSize atomic.Int64
	maxSize atomic.Int64
}

// NewWriteAsyncer 创建新的异步写入器
// NewWriteAsyncer creates a new asynchronous writer
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

	wa.wg.Add(1) // 只需要一个goroutine，减少了一个goroutine
	go wa.poller()

	return wa
}

// Stop 停止异步写入器
// Stop stops the asynchronous writer
func (wa *WriteAsyncer) Stop() {
	wa.once.Do(func() {
		wa.state.SetRunning(false)
		wa.cancel()
		wa.wg.Wait()
		wa.cleanQueueToWriter()
		wa.bufferedWriter.Flush()
		wa.bufferedWriter.Reset(io.Discard)
	})
}

// Write 实现写入方法
// Write implements the write method
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

	// 更新大小统计，用于优化预分配
	// Update size statistics for optimizing preallocation
	currentMax := wa.maxSize.Load()
	if int64(l) > currentMax {
		wa.maxSize.Store(int64(l))
	}

	// 使用指数移动平均计算平均大小
	// Use exponential moving average to calculate average size
	avgSize := wa.avgSize.Load()
	if avgSize == 0 {
		wa.avgSize.Store(int64(l))
	} else {
		// EMA = currentAvg * 0.8 + newValue * 0.2
		newAvg := (avgSize * 8 / 10) + (int64(l) * 2 / 10)
		wa.avgSize.Store(newAvg)
	}

	// 获取大小合适的缓冲区
	// Get a buffer of appropriate size
	buff := wa.bufferpool.Get()
	buff.Grow(l)

	if n, err = buff.Write(p); err != nil {
		wa.bufferpool.Put(buff)
		return 0, err
	}

	wa.config.queue.Push(buff)
	return l, nil
}

// flushBufferedWriter 刷新缓冲的写入器
// flushBufferedWriter flushes the buffered writer
func (wa *WriteAsyncer) flushBufferedWriter(content []byte) (int, error) {
	sizeOfContent := len(content)
	if sizeOfContent == 0 {
		return 0, nil
	}

	// 如果内容大小超过可用空间且缓冲区非空，则先刷新
	// If content size exceeds available space and buffer is not empty, flush first
	if sizeOfContent > wa.bufferedWriter.Available() && wa.bufferedWriter.Buffered() > 0 {
		if err := wa.bufferedWriter.Flush(); err != nil {
			return 0, err
		}
	}

	// 如果内容大小超过缓冲区大小，直接写入
	// If content size exceeds buffer size, write directly
	if sizeOfContent >= wa.config.buffSize {
		return wa.writer.Write(content)
	}

	return wa.bufferedWriter.Write(content)
}

// poller 轮询器，处理写入请求和心跳检查
// poller handles write requests and heartbeat checks
func (wa *WriteAsyncer) poller() {
	// 使用配置的心跳间隔，而不是硬编码值
	// Use the configured heartbeat interval instead of hardcoded value
	heartbeat := time.NewTicker(wa.config.heartbeatInterval)

	// 当前时间，减少系统调用
	// Current time, reducing system calls
	var now int64

	// 更新计时器时间，每秒更新一次
	// Update timer time, once per second
	timeUpdateTicker := time.NewTicker(time.Second)
	now = time.Now().UnixMilli() // 初始化当前时间 / Initialize current time
	wa.timer.Store(now)

	defer func() {
		timeUpdateTicker.Stop()
		heartbeat.Stop()
		wa.wg.Done()
	}()

	for {
		// 首先处理队列中的所有元素，优先级最高
		// First process all elements in the queue, highest priority
		for {
			if element := wa.config.queue.Pop(); element != nil {
				wa.executeFunc(element.(*bytes.Buffer))
				continue
			}
			break // 队列为空时退出内循环
		}

		// 使用select处理心跳和上下文
		// Use select to handle heartbeat and context
		select {
		case <-wa.ctx.Done():
			return

		case <-heartbeat.C:
			// 检查是否需要刷新缓冲区
			// Check if buffer needs to be flushed
			if wa.bufferedWriter.Buffered() > 0 {
				// 计算闲置时间
				// Calculate idle time
				now = wa.timer.Load()
				if (now - wa.state.GetExecuteAt()) >= wa.config.idleTimeout.Milliseconds() {
					if err := wa.bufferedWriter.Flush(); err != nil {
						wa.config.callback.OnWriteFailed(nil, err)
					}
					wa.state.SetExecuteAt(now)
				}
			}

		case <-timeUpdateTicker.C:
			// 更新当前时间
			// Update current time
			now = time.Now().UnixMilli()
			wa.timer.Store(now)
		}
	}
}

// executeFunc 执行写入操作
// executeFunc executes the write operation
func (wa *WriteAsyncer) executeFunc(buff *bytes.Buffer) {
	wa.state.SetExecuteAt(wa.timer.Load())
	content := buff.Bytes()

	if _, err := wa.flushBufferedWriter(content); err != nil {
		// 只在错误时复制内容，减少内存分配
		// Only copy content when there's an error, reducing memory allocation
		if wa.config.callback != nil {
			failContent := make([]byte, len(content))
			copy(failContent, content)
			wa.config.callback.OnWriteFailed(failContent, err)
		}
	}

	wa.bufferpool.Put(buff)
}

// cleanQueueToWriter 清理队列中的所有内容到写入器
// cleanQueueToWriter cleans all content in the queue to the writer
func (wa *WriteAsyncer) cleanQueueToWriter() {
	for {
		elem := wa.config.queue.Pop()
		if elem == nil {
			break
		}
		wa.executeFunc(elem.(*bytes.Buffer))
	}
}
