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

// WriteAsyncer 是一个异步写入器
// WriteAsyncer is an async writer
type WriteAsyncer struct {
	config         *Config            // 配置信息，包含各种参数
	queue          QueueInterface     // 队列接口，用于存储待写入的数据
	writer         io.Writer          // 写入器，实际执行写入操作的对象
	bufferedWriter *bufio.Writer      // 带缓冲的写入器，提高写入效率
	timer          atomic.Int64       // 计时器，用于定时检查队列
	once           sync.Once          // 一次性执行，用于确保某些操作只执行一次
	ctx            context.Context    // 上下文，用于控制异步操作的开始和结束
	cancel         context.CancelFunc // 取消函数，用于停止异步操作
	wg             sync.WaitGroup     // 等待组，用于等待异步操作完成
	state          Status             // 状态，记录 WriteAsyncer 的运行状态
	elementpool    *pool.Pool         // 元素池，用于复用 Element 对象，减少内存分配
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
		config:         conf,                                                                   // 配置信息
		queue:          lfq.NewLockFreeQueue(),                                                 // 创建无锁队列
		writer:         writer,                                                                 // 写入器
		bufferedWriter: bufio.NewWriterSize(writer, conf.buffsize),                             // 创建带缓冲的写入器
		state:          Status{},                                                               // 初始化状态
		elementpool:    pool.NewPool(func() any { return &Element{} }, lfs.NewLockFreeStack()), // 创建元素池
		timer:          atomic.Int64{},                                                         // 初始化计时器
		once:           sync.Once{},                                                            // 初始化一次性执行
		wg:             sync.WaitGroup{},                                                       // 初始化等待组
	}
	wa.ctx, wa.cancel = context.WithCancel(context.Background()) // 创建上下文和取消函数
	wa.state.executeAt.Store(time.Now().UnixMilli())             // 存储当前时间
	wa.state.running.Store(true)                                 // 设置运行状态为 true

	// 启动 poller 和 updateTimer
	// Start poller and updateTimer
	wa.wg.Add(2)        // 增加等待组的计数
	go wa.poller()      // 启动 poller
	go wa.updateTimer() // 启动 updateTimer

	return wa // 返回 WriteAsyncer 实例
}

// Stop 停止 WriteAsyncer
// Stop stops WriteAsyncer
func (wa *WriteAsyncer) Stop() {
	wa.once.Do(func() { // 确保以下操作只执行一次
		wa.state.running.Store(false) // 设置运行状态为 false
		wa.cancel()                   // 调用取消函数，停止所有异步操作
		wa.wg.Wait()                  // 等待所有异步操作完成
		wa.bufferedWriter.Flush()     // 清空缓冲区，将所有数据写入目标
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

	// executeFunc 是 poller 的执行函数
	// executeFunc is the execution function of the poller
	executeFunc := func(e *Element) {
		now := wa.timer.Load()        // 获取当前时间
		wa.state.executeAt.Store(now) // 存储当前时间

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
