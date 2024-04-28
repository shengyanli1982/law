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

// 定义默认的心跳间隔为 500 毫秒
// Define the default heartbeat interval as 500 milliseconds
const defaultHeartbeatInterval = 500 * time.Millisecond

// 定义默认的空闲超时为 5 秒
// Define the default idle timeout as 5 seconds
const defaultIdleTimeout = 5 * time.Second

// 定义一个错误，表示写异步器已经关闭
// Define an error indicating that the write asyncer is closed
var ErrorWriteAsyncerIsClosed = errors.New("write asyncer is closed")

// WriteAsyncer 结构体用于实现写异步器
// The WriteAsyncer struct is used to implement the write asyncer
type WriteAsyncer struct {
	// config 用于存储写异步器的配置
	// config is used to store the configuration of the write asyncer
	config *Config

	// writer 用于写入数据
	// writer is used to write data
	writer io.Writer

	// bufferedWriter 用于缓冲写入的数据
	// bufferedWriter is used to buffer the data to be written
	bufferedWriter *bufio.Writer

	// timer 用于控制写入的时间
	// timer is used to control the time of writing
	timer atomic.Int64

	// once 用于确保某个操作只执行一次
	// once is used to ensure that an operation is performed only once
	once sync.Once

	// ctx 用于控制写异步器的生命周期
	// ctx is used to control the lifecycle of the write asyncer
	ctx context.Context

	// cancel 用于取消写异步器的操作
	// cancel is used to cancel the operation of the write asyncer
	cancel context.CancelFunc

	// wg 用于等待写异步器的所有操作完成
	// wg is used to wait for all operations of the write asyncer to complete
	wg sync.WaitGroup

	// state 用于存储写异步器的状态
	// state is used to store the status of the write asyncer
	state *wr.Status

	// elementpool 用于存储元素池
	// elementpool is used to store the element pool
	elementpool *wr.ElementPool
}

// NewWriteAsyncer 函数用于创建一个新的 WriteAsyncer 实例
// The NewWriteAsyncer function is used to create a new WriteAsyncer instance
func NewWriteAsyncer(writer io.Writer, conf *Config) *WriteAsyncer {
	// 如果 writer 参数为 nil，那么将其设置为 os.Stdout
	// If the writer parameter is nil, then set it to os.Stdout
	if writer == nil {
		writer = os.Stdout
	}

	// 检查配置是否有效，如果无效则使用默认配置
	// Check if the configuration is valid, if not, use the default configuration
	conf = isConfigValid(conf)

	// 创建一个新的 WriteAsyncer 实例
	// Create a new WriteAsyncer instance
	wa := &WriteAsyncer{
		// 设置配置
		// Set the configuration
		config: conf,

		// 设置写入器
		// Set the writer
		writer: writer,

		// 创建一个新的带有指定缓冲区大小的 bufio.Writer 实例
		// Create a new bufio.Writer instance with the specified buffer size
		bufferedWriter: bufio.NewWriterSize(writer, conf.buffSize),

		// 初始化状态
		// Initialize the status
		state: wr.NewStatus(),

		// 初始化计时器
		// Initialize the timer
		timer: atomic.Int64{},

		// 初始化 once
		// Initialize once
		once: sync.Once{},

		// 初始化 wg
		// Initialize wg
		wg: sync.WaitGroup{},

		// 初始化元素池
		// Initialize the element pool
		elementpool: wr.NewElementPool(),
	}

	// 创建一个新的 context.Context 实例，并设置一个取消函数
	// Create a new context.Context instance and set a cancel function
	wa.ctx, wa.cancel = context.WithCancel(context.Background())

	// 设置下一次执行的时间为当前时间
	// Set the time of the next execution to the current time
	wa.state.SetExecuteAt(time.Now().UnixMilli())

	// 设置 running 为 true，表示 WriteAsyncer 正在运行
	// Set running to true, indicating that WriteAsyncer is running
	wa.state.SetRunning(true)

	// 增加 wg 的计数
	// Increase the count of wg
	wa.wg.Add(2)

	// 启动 poller 协程
	// Start the poller goroutine
	go wa.poller()

	// 启动 updateTimer 协程
	// Start the updateTimer goroutine
	go wa.updateTimer()

	// 返回新创建的 WriteAsyncer 实例
	// Return the newly created WriteAsyncer instance
	return wa
}

// Stop 方法用于停止 WriteAsyncer
// The Stop method is used to stop the WriteAsyncer
func (wa *WriteAsyncer) Stop() {
	// 使用 once.Do 方法确保以下的操作只执行一次
	// Use the once.Do method to ensure that the following operations are performed only once
	wa.once.Do(func() {
		// 将 running 状态设置为 false，表示 WriteAsyncer 已经停止
		// Set the running status to false, indicating that the WriteAsyncer has stopped
		wa.state.SetRunning(false)

		// 调用 cancel 函数取消 WriteAsyncer 的所有操作
		// Call the cancel function to cancel all operations of the WriteAsyncer
		wa.cancel()

		// 等待 WriteAsyncer 的所有操作完成
		// Wait for all operations of the WriteAsyncer to complete
		wa.wg.Wait()

		// 将队列中的所有数据写入到 writer
		// Write all data in the queue to the writer
		wa.cleanQueueToWriter()

		// 刷新 bufferedWriter，将所有缓冲的数据写入到 writer
		// Flush the bufferedWriter, writing all buffered data to the writer
		wa.bufferedWriter.Flush()

		// 重置 bufferedWriter，将其设置为 io.Discard
		// Reset the bufferedWriter, setting it to io.Discard
		wa.bufferedWriter.Reset(io.Discard)
	})
}

// Write 方法用于将数据写入到 WriteAsyncer
// The Write method is used to write data to the WriteAsyncer
func (wa *WriteAsyncer) Write(p []byte) (n int, err error) {
	// 如果 WriteAsyncer 已经停止，那么返回错误
	// If the WriteAsyncer has stopped, then return an error
	if !wa.state.IsRunning() {
		return 0, ErrorWriteAsyncerIsClosed
	}

	// 从元素池中获取一个元素
	// Get an elem from the elem pool
	elem := wa.elementpool.Get()

	// 将数据设置到元素的 buffer 字段
	// Set the data to the buffer field of the element
	if n, err = elem.Write(p); err != nil {
		return
	}

	// 将元素添加到队列
	// Add the element to the queue
	wa.config.queue.Push(elem)

	// 返回数据的长度和 nil 错误
	// Return the length of the data and a nil error
	return len(p), nil
}

// flushBufferedWriter 方法用于将数据写入到 bufferedWriter
// The flushBufferedWriter method is used to write data to the bufferedWriter
func (wa *WriteAsyncer) flushBufferedWriter(p []byte) (int, error) {
	// 如果数据的长度大于 bufferedWriter 的可用空间，并且 bufferedWriter 中已经有缓冲的数据
	// If the length of the data is greater than the available space of the bufferedWriter, and there is already buffered data in the bufferedWriter
	if len(p) > wa.bufferedWriter.Available() && wa.bufferedWriter.Buffered() > 0 {
		// 刷新 bufferedWriter，将所有缓冲的数据写入到 writer
		// Flush the bufferedWriter, writing all buffered data to the writer
		if err := wa.bufferedWriter.Flush(); err != nil {
			// 如果刷新失败，那么直接将数据写入到 writer，并返回写入的长度和错误
			// If the flush fails, then write the data directly to the writer and return the length of the write and the error
			return wa.writer.Write(p)
		}
	}

	// 将数据写入到 bufferedWriter，并返回写入的长度和错误
	// Write the data to the bufferedWriter and return the length of the write and the error
	return wa.bufferedWriter.Write(p)
}

// poller 方法用于从队列中获取元素并执行相应的函数
// The poller method is used to get elements from the queue and execute the corresponding functions
func (wa *WriteAsyncer) poller() {
	// 创建一个新的定时器，用于定时检查队列
	// Create a new timer for periodically checking the queue
	heartbeat := time.NewTicker(defaultHeartbeatInterval)

	// 使用 defer 语句确保在函数结束时停止定时器并完成减少 WaitGroup 的计数
	// Use a defer statement to ensure that the timer is stopped and the count of WaitGroup is reduced when the function ends
	defer func() {
		heartbeat.Stop()
		wa.wg.Done()
	}()

	// 使用无限循环来不断从队列中获取元素
	// Use an infinite loop to continuously get elements from the queue
	for {
		// 尝试从队列中弹出一个元素
		// Try to pop an element from the queue
		elem := wa.config.queue.Pop()

		// 如果元素不为空，执行 executeFunc 函数
		// If the element is not null, execute the executeFunc function
		if elem != nil {
			wa.executeFunc(elem.(*bytes.Buffer))
		} else {
			select {
			// 如果接收到 ctx.Done 的信号，那么结束循环
			// If the ctx.Done signal is received, then end the loop
			case <-wa.ctx.Done():
				return

			// 如果等待了一段时间，那么检查 bufferedWriter 中是否有缓冲的数据并且已经超过了空闲超时时间
			// If a period of time has passed, then check whether there is buffered data in the bufferedWriter and it has exceeded the idle timeout
			case <-heartbeat.C:
				// 获取当前时间
				// Get the current time
				now := wa.timer.Load()

				// 计算当前时间与上次执行时间的差值
				// Calculate the difference between the current time and the last execution time
				diff := now - wa.state.GetExecuteAt()

				// 如果 bufferedWriter 中有缓冲的数据，并且已经超过了空闲超时时间
				// If there is buffered data in the bufferedWriter and it has exceeded the idle timeout
				if wa.bufferedWriter.Buffered() > 0 && diff >= defaultIdleTimeout.Milliseconds() {
					// 刷新 bufferedWriter，将所有缓冲的数据写入到 writer
					// Flush the bufferedWriter, writing all buffered data to the writer
					if err := wa.bufferedWriter.Flush(); err != nil {
						// 如果在刷新 bufferedWriter 时发生错误，调用 OnWriteFailure 回调函数
						// If an error occurs while flushing the bufferedWriter, call the OnWriteFailure callback function
						wa.config.callback.OnWriteFailed(nil, err)
					}

					// 更新上次执行时间为当前时间
					// Update the last execution time to the current time
					wa.state.SetExecuteAt(now)
				}
			}
		}
	}
}

// updateTimer 方法用于更新 WriteAsyncer 的 timer 字段
// The updateTimer method is used to update the timer field of the WriteAsyncer
func (wa *WriteAsyncer) updateTimer() {
	// 创建一个每秒触发一次的定时器
	// Create a timer that triggers once per second
	ticker := time.NewTicker(time.Second)

	// 使用 defer 语句确保在函数返回时停止定时器并减少 WaitGroup 的计数
	// Use a defer statement to ensure that the timer is stopped and the WaitGroup count is decremented when the function returns
	defer func() {
		ticker.Stop()
		wa.wg.Done()
	}()

	// 使用无限循环来不断检查定时器和 ctx.Done 通道
	// Use an infinite loop to continuously check the timer and ctx.Done channel
	for {
		select {
		// 如果 ctx.Done 通道接收到数据，那么返回，结束这个函数
		// If the ctx.Done channel receives data, then return and end this function
		case <-wa.ctx.Done():
			return

		// 如果定时器触发，那么更新 timer 字段为当前的 Unix 毫秒时间
		// If the timer triggers, then update the timer field to the current Unix millisecond time
		case <-ticker.C:
			wa.timer.Store(time.Now().UnixMilli())
		}
	}
}

// executeFunc 方法用于执行 WriteAsyncer 的写入操作
// The executeFunc method is used to perform the write operation of the WriteAsyncer
func (wa *WriteAsyncer) executeFunc(elem *bytes.Buffer) {
	// 获取当前的 Unix 毫秒时间
	// Get the current Unix millisecond time
	now := wa.timer.Load()

	// 更新上次执行时间为当前时间
	// Update the last execution time to the current time
	wa.state.SetExecuteAt(now)

	// content 是一个变量，它获取 elem 的缓冲区的字节
	// content is a variable that gets the bytes of the buffer of elem
	content := elem.Bytes()

	// 将元素的数据写入到 bufferedWriter
	// Write the data of the element to the bufferedWriter
	if _, err := wa.flushBufferedWriter(content); err != nil {
		// 如果写入失败，那么将 content 复制到一个新的切片中。因为 Buffer 会被重置，原有的数据会被覆盖。
		// If the write fails, then copy content to a new slice. Because the Buffer will be reset, the original data will be overwritten.
		failContent := make([]byte, len(content))
		copy(failContent, content)

		// 如果写入失败，调用回调函数 OnWriteFailure
		// If the write fails, call the callback function OnWriteFailure
		wa.config.callback.OnWriteFailed(failContent, err)
	}

	// 将 elem 放回到 elementpool 中
	// Put elem back into the elementpool
	wa.elementpool.Put(elem)
}

// cleanQueueToWriter 方法用于将队列中的所有数据写入到 writer
// The cleanQueueToWriter method is used to write all data in the queue to the writer
func (wa *WriteAsyncer) cleanQueueToWriter() {
	// 使用无限循环来不断从队列中取出元素并执行写入操作
	// Use an infinite loop to continuously take elements from the queue and perform write operations
	for {
		// 从队列中取出一个元素
		// Take an element from the queue
		elem := wa.config.queue.Pop()

		// 如果元素为 nil，那么跳出循环
		// If the element is nil, then break the loop
		if elem == nil {
			break
		}

		// 执行写入操作
		// Perform the write operation
		wa.executeFunc(elem.(*bytes.Buffer))
	}
}
