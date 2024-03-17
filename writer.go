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
	buffer   []byte // 缓冲区，存储数据
	updateAt int64  // 更新时间，记录元素最后一次被修改的时间
}

// Reset 重置元素
// Reset resets the element
func (e *Element) Reset() {
	e.buffer = nil // 清空缓冲区
	e.updateAt = 0 // 重置更新时间
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

// WriteAsyncer 是一个异步写入器，它使用一个无锁队列和一个带缓冲的写入器来实现异步写入
// WriteAsyncer is an async writer that uses a lock-free queue and a buffered writer to implement async writing
type WriteAsyncer struct {
	// config 是 WriteAsyncer 的配置信息，包含各种参数
	// config is the configuration information of WriteAsyncer, containing various parameters
	config *Config

	// queue 是一个队列接口，用于存储待写入的数据
	// queue is a queue interface used to store data to be written
	queue QueueInterface

	// writer 是实际执行写入操作的对象
	// writer is the object that actually performs the write operation
	writer io.Writer

	// bufferedWriter 是一个带缓冲的写入器，它可以减少实际的 IO 操作次数
	// bufferedWriter is a buffered writer that can reduce the actual number of IO operations
	bufferedWriter *bufio.Writer

	// timer 是一个原子整数，用于存储当前的时间戳
	// timer is an atomic integer used to store the current timestamp
	timer atomic.Int64

	// once 是一个 sync.Once 对象，用于确保某些操作只执行一次
	// once is a sync.Once object used to ensure that certain operations are only performed once
	once sync.Once

	// ctx 是一个 context.Context 对象，用于控制 WriteAsyncer 的生命周期
	// ctx is a context.Context object used to control the lifecycle of WriteAsyncer
	ctx context.Context

	// cancel 是一个取消函数，用于取消 ctx
	// cancel is a cancel function used to cancel ctx
	cancel context.CancelFunc

	// wg 是一个 sync.WaitGroup 对象，用于等待所有的 goroutine 结束
	// wg is a sync.WaitGroup object used to wait for all goroutines to end
	wg sync.WaitGroup

	// state 是 WriteAsyncer 的状态，包含运行状态和最后执行时间
	// state is the status of WriteAsyncer, including running status and last execution time
	state Status

	// elementpool 是一个元素池，用于存储和复用 Element 对象
	// elementpool is an element pool used to store and reuse Element objects
	elementpool *pool.Pool
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
		// 设置配置
		// Set configuration
		config: conf,

		// 创建无锁队列
		// Create lock-free queue
		queue: lfq.NewLockFreeQueue(),

		// 设置写入器
		// Set writer
		writer: writer,

		// 创建带缓冲的写入器
		// Create buffered writer
		bufferedWriter: bufio.NewWriterSize(writer, conf.buffsize),

		// 初始化状态
		// Initialize status
		state: Status{},

		// 创建元素池
		// Create element pool
		elementpool: pool.NewPool(func() any { return &Element{} }, lfs.NewLockFreeStack()),

		// 初始化计时器
		// Initialize timer
		timer: atomic.Int64{},

		// 初始化 sync.Once
		// Initialize sync.Once
		once: sync.Once{},

		// 初始化 WaitGroup
		// Initialize WaitGroup
		wg: sync.WaitGroup{},
	}

	// 创建带取消功能的 context
	// Create a context with cancellation
	wa.ctx, wa.cancel = context.WithCancel(context.Background())

	// 设置执行时间
	// Set execution time
	wa.state.executeAt.Store(time.Now().UnixMilli())

	// 设置运行状态为 true
	// Set running status to true
	wa.state.running.Store(true)

	// 增加 WaitGroup 的计数
	// Increase the count of WaitGroup
	wa.wg.Add(2)

	// 启动轮询器
	// Start the poller
	go wa.poller()

	// 启动计时器更新器
	// Start the timer updater
	go wa.updateTimer()

	// 返回 WriteAsyncer 实例
	// Return the WriteAsyncer instance
	return wa
}

// Stop 停止 WriteAsyncer
// Stop stops WriteAsyncer
func (wa *WriteAsyncer) Stop() {
	// 使用 sync.Once 确保 Stop 方法只被执行一次
	// Use sync.Once to ensure that the Stop method is only executed once
	wa.once.Do(func() {
		// 将 running 状态设为 false，表示 WriteAsyncer 已经停止
		// Set the running state to false, indicating that WriteAsyncer has stopped
		wa.state.running.Store(false)

		// 调用 cancel 函数，取消所有的 context
		// Call the cancel function to cancel all contexts
		wa.cancel()

		// 等待所有的 goroutine 结束
		// Wait for all goroutines to end
		wa.wg.Wait()

		// 清理队列中的所有元素，将它们写入到 Writer
		// Clean up all elements in the queue and write them to Writer
		wa.cleaningQueueToWriter()

		// 刷新 bufferedWriter 中的所有数据
		// Flush all data in bufferedWriter
		wa.bufferedWriter.Flush()
	})
}

// Write 实现了 io.Writer 接口，用于将数据写入到 WriteAsyncer
// Write implements the io.Writer interface, used to write data to WriteAsyncer
func (wa *WriteAsyncer) Write(p []byte) (n int, err error) {
	// 如果 WriteAsyncer 已经关闭，就返回错误
	// If WriteAsyncer has been closed, return an error
	if !wa.state.running.Load() {
		return 0, ErrorWriteAsyncerIsClosed
	}

	// 从资源池中获取元素，并更新元素数据
	// Get the element from the resource pool and update the element data
	element := wa.elementpool.Get().(*Element)

	// 更新元素的缓冲区数据
	// Update the buffer data of the element
	element.buffer = p

	// 更新元素的时间戳
	// Update the timestamp of the element
	element.updateAt = wa.timer.Load()

	// 将数据写入到队列中
	// Write data to the queue
	wa.queue.Push(element)

	// 调用回调方法，通知数据已经被推入队列
	// Call the callback method to notify that the data has been pushed into the queue
	wa.config.callback.OnPushQueue(p)

	// 返回写入的数据长度和 nil 错误
	// Return the length of the written data and nil error
	return len(p), nil
}

// flushBufferedWriter 将缓冲区中的数据写入到底层 io.Writer
// flushBufferedWriter writes the data in the buffer to the underlying io.Writer
func (wa *WriteAsyncer) flushBufferedWriter(p []byte) (int, error) {
	// 调用回调方法
	// Call the callback method
	wa.config.callback.OnWrite(p)

	// 如果缓冲区可用空间不足，并且缓冲区中有数据，则先将缓冲区中的数据写入目标
	// If the available space in the buffer is not enough and there is data in the buffer, write the data in the buffer to the target first
	if len(p) > wa.bufferedWriter.Available() && wa.bufferedWriter.Buffered() > 0 {
		if err := wa.bufferedWriter.Flush(); err != nil {
			// 如果 Flush 失败，则直接将数据写入目标
			// If Flush fails, write the data directly to the target
			return wa.writer.Write(p)
		}
	}
	// 将数据写入缓冲区
	// Write data to the buffer
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
		heartbeat.Stop() // 停止心跳检测
		wa.wg.Done()     // 减少等待组的计数
	}()

	// 循环检查队列中的数据
	// Loop to check the data in the queue
	for {
		select {
		case <-wa.ctx.Done():
			// 如果收到了 ctx 的 Done 信号，就退出
			// If the Done signal of ctx is received, exit
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
				wa.executeFunc(elem.(*Element))
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
						// 如果 flush bufferedWriter 出错，就记录错误日志
						// If flushing bufferedWriter fails, log the error
						wa.config.logger.Errorf("buffered writer flush error, error: %s", err.Error())
					}

					// 更新 executeAt 的时间
					// Update the time of executeAt
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
	// 创建一个每秒触发一次的定时器
	// Create a timer that triggers once per second
	ticker := time.NewTicker(time.Second)

	// 使用 defer 语句确保在函数退出时停止定时器并减少等待组的计数
	// Use a defer statement to ensure that the timer is stopped and the wait group count is decreased when the function exits
	defer func() {
		ticker.Stop() // 停止定时器
		wa.wg.Done()  // 减少等待组的计数
	}()

	// 使用无限循环来持续更新时间戳
	// Use an infinite loop to continuously update the timestamp
	for {
		select {
		case <-wa.ctx.Done(): // 如果收到上下文的 Done 信号，就退出循环
			return
		case <-ticker.C: // 如果定时器触发，就更新时间戳
			// 使用当前的 Unix 毫秒时间戳更新 timer
			// Update timer with the current Unix millisecond timestamp
			wa.timer.Store(time.Now().UnixMilli())
		}
	}
}

// executeFunc 是 poller 的执行函数
// executeFunc is the execution function of the poller
func (wa *WriteAsyncer) executeFunc(elem *Element) {
	now := wa.timer.Load()        // 获取当前时间
	wa.state.executeAt.Store(now) // 存储当前时间

	// 调用回调方法
	// Call the callback method
	wa.config.callback.OnPopQueue(elem.buffer, now-elem.updateAt)

	// 将 buffer 中的内容写入到底层 io.Writer，如果有错误就是调用 logger 的 Errorf 方法
	// Write the content in the buffer to the underlying io.Writer, if there is an error, call the Errorf method of logger
	if _, err := wa.flushBufferedWriter(elem.buffer); err != nil {
		wa.config.logger.Errorf("data write error, error: %s, message: %s", err.Error(), util.BytesToString(elem.buffer))
	}

	// 重置元素，并归还资源池
	// Reset the element and return it to the resource pool
	elem.Reset()
	wa.elementpool.Put(elem)
}

// cleaningQueueToWriter 是一个方法，用于清理队列中的所有元素，并将它们写入到底层的 io.Writer
// cleaningQueueToWriter is a method that cleans up all elements in the queue and writes them to the underlying io.Writer
func (wa *WriteAsyncer) cleaningQueueToWriter() {
	// 循环处理队列中的所有元素
	// Process all elements in the queue in a loop
	for {
		// 从队列中取出一个元素
		// Pop an element from the queue
		elem := wa.queue.Pop()

		// 如果元素为空，就退出循环
		// If the element is nil, break the loop
		if elem == nil {
			break
		}

		// 将元素转换为 *Element 类型，并执行 executeFunc 方法
		// Convert the element to *Element type and execute the executeFunc method
		wa.executeFunc(elem.(*Element))
	}
}
