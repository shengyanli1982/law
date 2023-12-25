package law

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	buf "github.com/shengyanli1982/law/internal/buffer"
	list "github.com/shengyanli1982/law/internal/stl/deque"
	"github.com/shengyanli1982/law/internal/util"
)

const (
	defaultIdleTimeout = 5 * time.Second // 默认空闲时间为5秒 (Default idle timeout is 5 seconds)
)

type Config struct {
	bfsize int      // 缓冲区大小 (Buffer size)
	cb     Callback // 回调函数 (Callback function)
}

// 创建一个配置对象 (Create a new Config object)
func NewConfig() *Config {
	return &Config{bfsize: 0, cb: &emptyCallback{}}
}

// 设置缓冲区大小 (Set the buffer size)
func (c *Config) WithBufferSize(size int) *Config {
	c.bfsize = size
	return c
}

// 设置回调函数 (Set the callback function)
func (c *Config) WithCallback(cb Callback) *Config {
	c.cb = cb
	return c
}

type WriteAsyncer struct {
	closed         atomic.Bool          // 标识写入器是否关闭 (Indicates whether the writer is closed)
	bufferPool     *buf.ExtraBufferPool // 缓冲区池 (Buffer pool)
	writer         io.Writer            // 底层的写入器 (Underlying writer)
	bufferIoWriter *bufio.Writer        // 缓冲IO写入器 (Buffered IO writer)
	bufferIoLock   sync.Mutex           // 缓冲IO写入器锁 (Lock for buffered IO writer)
	listNodePool   *list.ListNodePool   // 链表节点池 (List node pool)
	once           sync.Once            // 用于确保只执行一次的同步对象 (Synchronization object to ensure only one execution)
	stopCtx        context.Context      // 停止上下文 (Stop context)
	stopCancel     context.CancelFunc   // 停止取消函数 (Stop cancel function)
	wg             sync.WaitGroup       // 等待组 (Wait group)
	queue          *list.Deque          // 队列 (Queue)
	queueLock      sync.Mutex           // 队列锁 (Queue lock)
	config         *Config              // 配置 (Configuration)
	idleAt         atomic.Int64         // 空闲时间戳 (Idle timestamp)
}

// 创建一个异步 writer (Create a new asynchronous writer)
func NewWriteAsyncer(w io.Writer, conf *Config) *WriteAsyncer {
	if w == nil {
		w = os.Stdout
	}

	wa := &WriteAsyncer{
		closed:         atomic.Bool{},
		writer:         w,
		bufferIoWriter: bufio.NewWriterSize(w, buf.DefaultBufferSize),
		bufferIoLock:   sync.Mutex{},
		queue:          list.NewDeque(),
		queueLock:      sync.Mutex{},
		once:           sync.Once{},
		wg:             sync.WaitGroup{},
		config:         conf,
		idleAt:         atomic.Int64{},
	}

	wa.isConfigValid()

	wa.stopCtx, wa.stopCancel = context.WithCancel(context.Background())
	wa.bufferPool = buf.NewExtraBufferPool(wa.config.bfsize)
	wa.listNodePool = list.NewListNodePool()
	wa.idleAt.Store(time.Now().UnixMilli())

	wa.wg.Add(2)
	go wa.poller()
	go wa.bufferIoWriterRefresh()

	return wa
}

// 创建一个默认的异步 writer (Create a new asynchronous writer with default config)
func DefaultWriteAsyncer() Writer {
	return NewWriteAsyncer(os.Stdout, nil)
}

// 将异步 writer 包装成 Writer (Wrap the asynchronous writer into Writer interface)
func WriteAsyncerWrapper(w *WriteAsyncer) Writer {
	return w
}

// 检查配置是否有效，如果无效，设置默认值 (Check if the config is valid, if not, set the default value)
func (wa *WriteAsyncer) isConfigValid() {
	if wa.config == nil {
		wa.config = NewConfig().WithBufferSize(buf.DefaultBufferSize).WithCallback(&emptyCallback{})
	} else {
		if wa.config.bfsize <= buf.DefaultBufferSize {
			wa.config.bfsize = buf.DefaultBufferSize
		}
		if wa.config.cb == nil {
			wa.config.cb = &emptyCallback{}
		}
	}
}

// 关闭写入器，停止刷新缓冲区和写入队列 (Stop the writer, flush the buffer and write queue)
func (wa *WriteAsyncer) Stop() {
	wa.once.Do(func() {
		wa.closed.Store(true)
		wa.stopCancel()
		wa.wg.Wait()
		wa.bufferIoLock.Lock()
		wa.bufferIoWriter.Flush()
		wa.bufferIoLock.Unlock()
	})
}

// 从队列中取出日志，写入到底层的 writer 中 (Poll log entries from the queue and write to the underlying writer)
func (wa *WriteAsyncer) poller() {
	heartbeat := time.NewTicker(time.Millisecond * 500) // 心跳 (Heartbeat)

	defer func() {
		heartbeat.Stop()
		wa.wg.Done()
	}()

	// 执行读取从链表中读取，并写入到底层的 writer 中 (Execute reading from the list and write to the underlying writer)
	exec := func(ln *list.Node) {
		eb := ln.Data().(*buf.ExtraBuffer)                // 从链表节点中获取缓冲区 (Get buffer from the list node)
		bytes := eb.Buffer().Bytes()                      // 从缓冲区中获取日志 (Get log entries from the buffer)
		now := time.Now().UnixMilli()                     // 获取当前时间戳 (Get current timestamp)
		wa.config.cb.OnPopQueue(bytes, now-eb.UpdateAt()) // 回调函数 (Callback function)

		wa.bufferIoLock.Lock()
		_, err := wa.buffWriter(bytes) // 将日志写入到底层的 writer 中 (Write log entries to the underlying writer)
		wa.bufferIoLock.Unlock()

		if err != nil { // 如果写入失败，则打印日志 (If write failed, print log)
			log.Printf("data write error, error: %s, message: %s", err.Error(), util.BytesToString(bytes))
		}

		wa.idleAt.Store(now)    // 设置空闲时间 (Set idle time)
		wa.listNodePool.Put(ln) // 将链表节点放回链表节点池中 (Put list node back into the list node pool)
		wa.bufferPool.Put(eb)   // 将缓冲区放回缓冲区池中 (Put buffer back into the buffer pool)
	}

	// 循环处理 (Loop processing)
	for {
		select {
		case <-wa.stopCtx.Done(): // 如果停止上下文被取消，则返回 (If the stop context is canceled, return)
			wa.queueLock.Lock()
			for {
				ln := wa.queue.Pop() // 从队列中取出链表节点 (Pop list node from the queue)
				if ln == nil {       // 如果链表节点为空，则退出 (If the list node is empty, exit)
					break
				}
				wa.queueLock.Unlock()
				exec(ln) // 执行 (Execute)
				wa.queueLock.Lock()
			}
			wa.queueLock.Unlock()
			return
		default: // 如果队列不为空，则从队列中取出数据 (If the queue is not empty, poll log entries from the queue)
			wa.queueLock.Lock()
			ln := wa.queue.Pop() // 从队列中取出链表节点 (Pop list node from the queue)
			wa.queueLock.Unlock()
			if ln != nil { // 如果链表节点不为空，则执行 (If the list node is not empty, execute)
				exec(ln) // 执行 (Execute)
			} else {
				<-heartbeat.C // 如果队列为空，则等待心跳 (If the queue is empty, wait for heartbeat)
			}
		}
	}
}

// 刷新缓冲区 (Flush the buffer)
func (wa *WriteAsyncer) bufferIoWriterRefresh() {
	heartbeat := time.NewTicker(time.Second) // 心跳 (Heartbeat)

	defer func() {
		heartbeat.Stop()
		wa.wg.Done()
	}()

	// 循环处理 (Loop processing)
	for {
		select {
		case <-wa.stopCtx.Done():
			return
		case <-heartbeat.C:
			wa.bufferIoLock.Lock()
			// 如果缓冲区有数据，并且空闲时间超过默认空闲时间，则刷新缓冲区 (If the buffer has data and the idle time exceeds the default idle time, flush the buffer)
			if wa.bufferIoWriter.Buffered() > 0 && time.Now().UnixMilli()-wa.idleAt.Load() > defaultIdleTimeout.Milliseconds() {
				if err := wa.bufferIoWriter.Flush(); err != nil { // 刷新缓冲区 (Flush the buffer)
					log.Printf("buffer io writer flush error, error: %s", err.Error())
				}
				wa.idleAt.Store(time.Now().UnixMilli()) // 设置空闲时间 (Set idle time)
			}
			wa.bufferIoLock.Unlock()
		}
	}
}

// 缓冲写入器 (Buffered writer)
func (wa *WriteAsyncer) buffWriter(p []byte) (int, error) {
	wa.config.cb.OnWrite(p)                                                         // 回调函数 (Callback function)
	if len(p) > wa.bufferIoWriter.Available() && wa.bufferIoWriter.Buffered() > 0 { // 如果日志长度大于缓冲区可用长度，并且缓冲区有数据 (If the log length is greater than the available length of the buffer and the buffer has data)
		if err := wa.bufferIoWriter.Flush(); err != nil { // 刷新缓冲区 (Flush the buffer)
			return wa.writer.Write(p) // 将日志写入到底层的 writer 中 (Write log entries to the underlying writer)
		}
	}
	return wa.bufferIoWriter.Write(p) // 将日志写入到缓冲区 (Write log entries to the buffer)
}

// 将日志写入到队列中 (Push log entries into the queue)
func (wa *WriteAsyncer) Write(p []byte) (int, error) {
	if wa.closed.Load() {
		return 0, ErrorWriterIsClosed
	}

	eb := wa.bufferPool.Get()              // 从缓冲区池中获取缓冲区 (Get buffer from the buffer pool)
	eb.Buffer().Write(p)                   // 将日志写入到缓冲区 (Write log entries into the buffer)
	eb.SetUpdateAt(time.Now().UnixMilli()) // 设置更新时间 (Set update time)
	ln := wa.listNodePool.Get()            // 从链表节点池中获取链表节点 (Get list node from the list node pool)
	ln.SetData(eb)                         // 设置链表节点数据 (Set list node data)

	wa.config.cb.OnPushQueue(p) // 回调函数 (Callback function)

	wa.queueLock.Lock()
	wa.queue.Push(ln) // 将链表节点放入队列 (Put list node into the queue)
	wa.queueLock.Unlock()

	return len(p), nil
}
