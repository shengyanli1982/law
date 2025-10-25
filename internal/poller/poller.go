package poller

import (
	"bufio"
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"time"

	wr "github.com/shengyanli1982/law/internal/writer"
)

// Queue 定义了队列接口
type Queue interface {
	Push(value interface{})
	Pop() interface{}
}

// Callback 定义了回调接口
type Callback interface {
	OnWriteFailed(content []byte, reason error)
}

// Poller 轮询器，负责异步处理队列中的写入请求
type Poller struct {
	queue             Queue          // 无锁队列
	writer            *bufio.Writer  // 带缓冲的写入器
	callback          Callback       // 错误回调
	hasCallback       bool           // 缓存callback是否存在
	state             *wr.Status     // 状态管理器
	bufferpool        *wr.BufferPool // 缓冲池
	timer             *atomic.Int64  // 计时器（共享）
	heartbeatInterval time.Duration  // 心跳间隔
	idleTimeout       time.Duration  // 闲置超时
}

// Config Poller配置
type Config struct {
	Queue             Queue          // 队列实现
	Writer            *bufio.Writer  // 带缓冲的写入器
	Callback          Callback       // 错误回调
	State             *wr.Status     // 状态管理器
	BufferPool        *wr.BufferPool // 缓冲池
	Timer             *atomic.Int64  // 计时器
	HeartbeatInterval time.Duration  // 心跳间隔
	IdleTimeout       time.Duration  // 闲置超时
}

// NewPoller 创建新的轮询器
func NewPoller(cfg *Config) *Poller {
	return &Poller{
		queue:             cfg.Queue,
		writer:            cfg.Writer,
		callback:          cfg.Callback,
		hasCallback:       cfg.Callback != nil,
		state:             cfg.State,
		bufferpool:        cfg.BufferPool,
		timer:             cfg.Timer,
		heartbeatInterval: cfg.HeartbeatInterval,
		idleTimeout:       cfg.IdleTimeout,
	}
}

// Run 启动轮询器，处理写入请求和心跳检查
func (p *Poller) Run(ctx context.Context, wg *sync.WaitGroup) {
	// 使用单一ticker，合并心跳检查和时间更新逻辑
	ticker := time.NewTicker(p.heartbeatInterval)
	var tickCount int64

	// 初始化计时器
	now := time.Now().UnixMilli()
	p.timer.Store(now)

	defer func() {
		ticker.Stop()
		wg.Done()
	}()

	for {
		// 首先处理队列中的所有元素，优先级最高
		for {
			if element := p.queue.Pop(); element != nil {
				p.executeFunc(element.(*bytes.Buffer))
				continue
			}
			break // 队列为空时退出内循环
		}

		// 使用select处理ticker和上下文
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			tickCount++

			// 每1秒更新时间缓存（heartbeatInterval为500ms时，每2次tick更新）
			// 这样可以减少time.Now()的系统调用开销
			if tickCount%(int64(time.Second/p.heartbeatInterval)) == 0 {
				now = time.Now().UnixMilli()
				p.timer.Store(now)
			}

			// 心跳检查：如果缓冲区有数据且超过空闲超时时间，则刷新
			if p.writer.Buffered() > 0 {
				cachedNow := p.timer.Load()
				if (cachedNow - p.state.GetExecuteAt()) >= p.idleTimeout.Milliseconds() {
					if err := p.writer.Flush(); err != nil {
						if p.hasCallback {
							p.callback.OnWriteFailed(nil, err)
						}
					}
					p.state.SetExecuteAt(cachedNow)
				}
			}
		}
	}
}

// executeFunc 执行写入操作
func (p *Poller) executeFunc(buff *bytes.Buffer) {
	p.state.SetExecuteAt(p.timer.Load())
	content := buff.Bytes()

	if _, err := p.flushBufferedWriter(content); err != nil {
		// 只在错误发生且回调函数存在时调用回调
		if p.hasCallback {
			// 直接使用原始buffer的内容作为回调参数，避免额外的内存分配
			p.callback.OnWriteFailed(content, err)
		}
	}

	p.bufferpool.Put(buff)
}

// flushBufferedWriter 刷新缓冲的写入器
func (p *Poller) flushBufferedWriter(content []byte) (int, error) {
	sizeOfContent := len(content)
	if sizeOfContent == 0 {
		return 0, nil
	}

	// 如果内容大小超过可用空间且缓冲区非空，则先刷新
	if sizeOfContent > p.writer.Available() && p.writer.Buffered() > 0 {
		if err := p.writer.Flush(); err != nil {
			return 0, err
		}
	}

	// 如果内容大小超过缓冲区大小，直接写入底层writer
	// 注意：这里需要通过反射或其他方式获取底层writer，暂时简化处理
	// 实际上bufio.Writer没有直接访问底层writer的方法
	// 我们依赖bufio.Writer的智能处理
	return p.writer.Write(content)
}

// CleanQueue 清理队列中的所有内容
func (p *Poller) CleanQueue() {
	for {
		elem := p.queue.Pop()
		if elem == nil {
			break
		}
		p.executeFunc(elem.(*bytes.Buffer))
	}
}
