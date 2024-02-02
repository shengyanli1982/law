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

	"github.com/shengyanli1982/law/internal/pool"
	lfq "github.com/shengyanli1982/law/internal/queue"
	lfs "github.com/shengyanli1982/law/internal/stack"
	"github.com/shengyanli1982/law/internal/util"
)

const (
	// defaultHeartbeatInterval 是 poller 检查队列的默认间隔
	// defaultHeartbeatInterval is the default interval for the poller to check the queue
	defaultHeartbeatInterval = 500 * time.Millisecond

	// defaultIdleTimeout 是 poller 的默认空闲超时时间
	// defaultIdleTimeout is the default idle timeout for the poller
	defaultIdleTimeout = 5 * time.Second
)

// ErrorWriteAsyncerIsClosed 表示 WriteAsyncer 已经关闭
// ErrorWriteAsyncerIsClosed indicates that WriteAsyncer has been closed
var ErrorWriteAsyncerIsClosed = errors.New("write asyncer is closed")

// Element 是队列中的元素
// Element is the element in the queue
type Element struct {
	buffer   []byte
	updateAt int64
}

// Reset 重置元素
// Reset resets the element
func (e *Element) Reset() {
	e.buffer = nil
	e.updateAt = 0
}

// Status 是 WriteAsyncer 的状态
// Status is the status of WriteAsyncer
type Status struct {
	// running 表示 WriteAsyncer 是否在运行
	// running indicates whether WriteAsyncer is running
	running atomic.Bool

	// executeAt 表示上次执行的时间
	// executeAt indicates the last time of execution
	executeAt atomic.Int64
}

// WriteAsyncer 是一个异步写入器
// WriteAsyncer is an async writer
type WriteAsyncer struct {
	config         *Config            // 配置信息
	queue          QueueInterface     // 队列接口
	writer         io.Writer          // 写入器
	bufferedWriter *bufio.Writer      // 带缓冲的写入器
	timer          atomic.Int64       // 计时器
	once           sync.Once          // 一次性执行
	ctx            context.Context    // 上下文
	cancel         context.CancelFunc // 取消函数
	wg             sync.WaitGroup     // 等待组
	state          Status             // 状态
	elementpool    *pool.Pool         // 元素池
}

// NewWriteAsyncer 返回一个 WriteAsyncer 实例
// NewWriteAsyncer returns an instance of WriteAsyncer
func NewWriteAsyncer(writer io.Writer, conf *Config) *WriteAsyncer {
	// 如果 writer 为空，则使用 os.Stdout
	// If writer is nil, use os.Stdout
	if writer == nil {
		writer = os.Stdout
	}

	// 判断 conf 配置内容是否有效
	// Check if the conf configuration is valid
	conf = isConfigValid(conf)

	// 初始化 WriteAsyncer 实例
	// Initialize the WriteAsyncer instance
	wa := &WriteAsyncer{
		config:         conf,
		queue:          lfq.NewLockFreeQueue(),
		writer:         writer,
		bufferedWriter: bufio.NewWriterSize(writer, conf.buffsize),
		state:          Status{},
		elementpool:    pool.NewPool(func() any { return &Element{} }, lfs.NewLockFreeStack()),
		timer:          atomic.Int64{},
		once:           sync.Once{},
		wg:             sync.WaitGroup{},
	}
	wa.ctx, wa.cancel = context.WithCancel(context.Background())
	wa.state.executeAt.Store(time.Now().UnixMilli())
	wa.state.running.Store(true)

	// 启动 poller 和 updateTimer
	// Start poller and updateTimer
	wa.wg.Add(2)
	go wa.poller()
	go wa.updateTimer()

	return wa
}

// Stop 停止 WriteAsyncer
// Stop stops WriteAsyncer
func (wa *WriteAsyncer) Stop() {
	wa.once.Do(func() {
		wa.state.running.Store(false)
		wa.cancel()
		wa.wg.Wait()
		wa.bufferedWriter.Flush()
	})
}

// Write 实现了 io.Writer 接口
// Write implements the io.Writer interface
func (wa *WriteAsyncer) Write(p []byte) (n int, err error) {
	// 如果 WriteAsyncer 已经关闭，就返回错误
	// If WriteAsyncer has been closed, return an error
	if !wa.state.running.Load() {
		return 0, ErrorWriteAsyncerIsClosed
	}

	// 从资源池中获取元素，并更新元素数据
	// Get the element from the resource pool and update the element data
	element := wa.elementpool.Get().(*Element)
	element.buffer = p
	element.updateAt = wa.timer.Load()

	// 将数据写入到队列中
	// Write data to the queue
	wa.queue.Push(element)

	// 调用回调方法
	// Call the callback method
	wa.config.callback.OnPushQueue(p)

	return len(p), nil
}

// flushBufferedWriter 将缓冲区中的数据写入到底层 io.Writer
// flushBufferedWriter writes the data in the buffer to the underlying io.Writer
func (wa *WriteAsyncer) flushBufferedWriter(p []byte) (int, error) {
	wa.config.callback.OnWrite(p)
	if len(p) > wa.bufferedWriter.Available() && wa.bufferedWriter.Buffered() > 0 {
		if err := wa.bufferedWriter.Flush(); err != nil {
			return wa.writer.Write(p)
		}
	}
	return wa.bufferedWriter.Write(p)
}

// poller 是一个轮询器，用于检查队列中的数据并将其写入到底层 io.Writer
// poller is a poller that checks the data in the queue and writes it to the underlying io.Writer
func (wa *WriteAsyncer) poller() {
	// 启动心跳检测
	// Start heartbeat detection
	heartbeat := time.NewTicker(defaultHeartbeatInterval)

	// 定义退出机制
	// Define the exit mechanism
	defer func() {
		heartbeat.Stop()
		wa.wg.Done()
	}()

	// executeFunc 是 poller 的执行函数
	// executeFunc is the execution function of the poller
	executeFunc := func(e *Element) {
		now := wa.timer.Load()
		wa.state.executeAt.Store(now)

		// 调用回调方法
		// Call the callback method
		wa.config.callback.OnPopQueue(e.buffer, now-e.updateAt)

		// 将 buffer 中的内容写入到底层 io.Writer，如果有错误就是调用 logger 的 Errorf 方法
		// Write the content in the buffer to the underlying io.Writer, if there is an error, call the Errorf method of logger
		if _, err := wa.flushBufferedWriter(e.buffer); err != nil {
			wa.config.logger.Errorf("data write error, error: %s, message: %s", err.Error(), util.BytesToString(e.buffer))
		}

		// 重置元素，并归还资源池
		// Reset the element and return it to the resource pool
		e.Reset()
		wa.elementpool.Put(e)
	}

	// 循环检查队列中的数据
	// Loop to check the data in the queue
	for {
		select {
		case <-wa.ctx.Done():
			// 如果 ctx 被取消，就将队列中的数据全部写入到底层 io.Writer
			// If ctx is canceled, write all the data in the queue to the underlying io.Writer
			for {
				elem := wa.queue.Pop()
				if elem == nil {
					break
				}
				executeFunc(elem.(*Element))
			}
			return

		default:
			// 如果 WriteAsyncer 已经关闭，就退出
			// If WriteAsyncer has been closed, exit
			if !wa.state.running.Load() {
				return
			}

			// 从队列中取出元素，如果元素不为空就执行 executeFunc。
			// 否则 sleep 一段时间，判断 bufferedWriter 是否有数据，如果有就 flush。
			// 随后判断是否需要 prune elementpool。
			// Pop an element from the queue, if the element is not empty, execute executeFunc.
			// Otherwise, sleep for a period of time, check if bufferedWriter has data, if so, flush.
			// Then check if elementpool needs to be pruned.
			elem := wa.queue.Pop()
			if elem != nil {
				// 如果元素不为空就执行 executeFunc
				// If the element is not empty, execute executeFunc
				executeFunc(elem.(*Element))
			} else {
				// 如果没有元素，就等待心跳信号
				// If there is no element, wait for the heartbeat signal
				<-heartbeat.C

				// 获取当前时间戳，计算 diff
				// Get the current timestamp and calculate diff
				now := wa.timer.Load()
				diff := now - wa.state.executeAt.Load()

				// 如果 bufferedWriter 中有数据，并且 diff 大于默认的空闲超时时间，就 flush bufferedWriter
				// If there is data in bufferedWriter and diff is greater than the default idle timeout, flush bufferedWriter
				if wa.bufferedWriter.Buffered() > 0 && diff > defaultIdleTimeout.Milliseconds() {
					if err := wa.bufferedWriter.Flush(); err != nil {
						wa.config.logger.Errorf("buffered writer flush error, error: %s", err.Error())
					}
					wa.state.executeAt.Store(now)
				}

				// 如果 diff 大于默认的空闲超时时间的 6 倍，就 prune elementpool
				// If diff is greater than 6 times the default idle timeout, prune elementpool
				if diff > defaultIdleTimeout.Milliseconds()*6 {
					wa.elementpool.Prune()
				}
			}
		}
	}
}

// updateTimer 是一个定时器，用于更新 WriteAsyncer 的时间戳
// updateTimer is a timer that updates the timestamp of WriteAsyncer
func (wa *WriteAsyncer) updateTimer() {
	ticker := time.NewTicker(time.Second)

	defer func() {
		ticker.Stop()
		wa.wg.Done()
	}()

	for {
		select {
		case <-wa.ctx.Done():
			return
		case <-ticker.C:
			wa.timer.Store(time.Now().UnixMilli())
		}
	}
}
