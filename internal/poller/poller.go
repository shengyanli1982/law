package poller

import (
	"bufio"
	"bytes"
	"context"
	"sync"
	"time"

	wr "github.com/shengyanli1982/law/internal/writer"
)

// Queue 定义了内部轮询器使用的类型化队列接口。
type Queue[T any] interface {
	Push(value T)
	Pop() T
}

// popBatchSize 单次批量出队的节点数上限：
// 摊薄消费者每条数据的加锁成本，又避免单批长时间独占队列锁。
const popBatchSize = 64

// batchPopper 队列批量出队能力接口（能力探测，不修改 Queue 公共接口）。
type batchPopper interface {
	PopBatch(max int) []*bytes.Buffer
}

// batchPopperInto 批量出队（复用调用方缓冲，零批次分配）能力接口。
// 优先于 batchPopper 使用。
type batchPopperInto interface {
	PopBatchInto(dst []*bytes.Buffer, max int) []*bytes.Buffer
}

// notifySource 队列空→非空通知能力接口（能力探测，不修改 Queue 公共接口）。
type notifySource interface {
	NotifyChan() <-chan struct{}
}

// Callback 定义了回调接口。
type Callback interface {
	OnWriteFailed(content []byte, reason error)
}

// Poller 轮询器，负责异步处理队列中的写入请求。
type Poller struct {
	queue             Queue[*bytes.Buffer]
	writer            *bufio.Writer
	callback          Callback
	hasCallback       bool
	bufferpool        *wr.BufferPool
	heartbeatInterval time.Duration
	idleTimeout       time.Duration
	// popBatchInto 批量出队（复用缓冲）能力；不具备时退回 popBatch，再退回逐条 Pop。
	popBatchInto func(dst []*bytes.Buffer, max int) []*bytes.Buffer
	// popBatch 批量出队能力；不具备时为 nil，回退逐条 Pop（兼容自定义 Queue）。
	popBatch func(max int) []*bytes.Buffer
	// batchBuf popBatchInto 的持久批次缓冲（容量 popBatchSize），避免每批分配。
	batchBuf []*bytes.Buffer
	// notifyCh 空→非空通知通道；不具备时为 nil，select 中该分支永不触发，
	// 等价于仅心跳驱动的轮询。
	notifyCh <-chan struct{}
}

// Config Poller配置。
type Config struct {
	Queue             Queue[*bytes.Buffer]
	Writer            *bufio.Writer
	Callback          Callback
	BufferPool        *wr.BufferPool
	HeartbeatInterval time.Duration
	IdleTimeout       time.Duration
}

// NewPoller 创建新的轮询器。
func NewPoller(cfg *Config) *Poller {
	p := &Poller{
		queue:             cfg.Queue,
		writer:            cfg.Writer,
		callback:          cfg.Callback,
		hasCallback:       cfg.Callback != nil,
		bufferpool:        cfg.BufferPool,
		heartbeatInterval: cfg.HeartbeatInterval,
		idleTimeout:       cfg.IdleTimeout,
	}

	// 能力探测：队列实现可选地提供批量出队与通知能力。
	// 批量出队优先 PopBatchInto（复用持久缓冲，零批次分配），退回 PopBatch。
	if bpi, ok := cfg.Queue.(batchPopperInto); ok {
		p.popBatchInto = bpi.PopBatchInto
		p.batchBuf = make([]*bytes.Buffer, 0, popBatchSize)
	} else if bp, ok := cfg.Queue.(batchPopper); ok {
		p.popBatch = bp.PopBatch
	}
	if ns, ok := cfg.Queue.(notifySource); ok {
		p.notifyCh = ns.NotifyChan()
	}

	return p
}

// drain 排空队列。优先批量出队复用缓冲（一次加锁摘取至多 popBatchSize
// 个节点、零批次分配），退回批量出队，再退回逐条 Pop（兼容自定义 Queue）。
// executeAt/lastFlushAt 通过指针回写，与调用方共享时钟状态。
func (p *Poller) drain(executeAt *int64, lastFlushAt *int64, nowMilli int64) {
	if p.popBatchInto != nil {
		for {
			p.batchBuf = p.popBatchInto(p.batchBuf[:0], popBatchSize)
			if len(p.batchBuf) == 0 {
				break
			}
			for i := range p.batchBuf {
				p.executeFunc(p.batchBuf[i], executeAt, lastFlushAt, nowMilli)
			}
		}
		return
	}

	if p.popBatch != nil {
		for {
			batch := p.popBatch(popBatchSize)
			if len(batch) == 0 {
				break
			}
			for i := range batch {
				p.executeFunc(batch[i], executeAt, lastFlushAt, nowMilli)
			}
		}
		return
	}

	for {
		element := p.queue.Pop()
		if element == nil {
			break
		}
		p.executeFunc(element, executeAt, lastFlushAt, nowMilli)
	}
}

// Run 启动轮询器，处理写入请求和心跳检查。
// 支持三种唤醒源：ctx 取消、心跳 ticker、队列空→非空通知（若队列具备）。
// 通知仅加速拾取，不改变 flush 判定（仍由心跳分支的双时钟驱动）。
func (p *Poller) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(p.heartbeatInterval)

	now := time.Now()
	nowMilli := now.UnixMilli()
	// executeAt 记录最近一次写入活动；lastFlushAt 记录最近一次成功落盘。
	// 两个时钟平行：持续写入时静默判据会被不断推迟，
	// 必须用 lastFlushAt 的有界判据兜底，防止 flush 饥饿。
	executeAt := nowMilli
	lastFlushAt := nowMilli

	defer func() {
		ticker.Stop()
		wg.Done()
	}()

	for {
		nowMilli = time.Now().UnixMilli()

		p.drain(&executeAt, &lastFlushAt, nowMilli)

		select {
		case <-ctx.Done():
			return

		case <-p.notifyCh:
			// 空→非空通知（若队列不具备则为 nil，永不触发）：
			// 仅触发新一轮 drain，不做 flush 判定。

		case <-ticker.C:
			nowMilli = time.Now().UnixMilli()

			if p.writer.Buffered() > 0 {
				idleMilli := p.idleTimeout.Milliseconds()
				if (nowMilli-executeAt) >= idleMilli || (nowMilli-lastFlushAt) >= idleMilli {
					if err := p.writer.Flush(); err != nil {
						if p.hasCallback {
							p.callback.OnWriteFailed(nil, err)
						}
					} else {
						executeAt = nowMilli
						lastFlushAt = nowMilli
					}
				}
			}
		}
	}
}

// executeFunc 执行写入操作。
func (p *Poller) executeFunc(buff *bytes.Buffer, executeAt *int64, lastFlushAt *int64, nowMilli int64) {
	*executeAt = nowMilli
	content := buff.Bytes()

	if _, err := p.flushBufferedWriter(content, lastFlushAt, nowMilli); err != nil {
		if p.hasCallback {
			p.callback.OnWriteFailed(content, err)
		}
	}

	p.bufferpool.Put(buff)
}

// flushBufferedWriter 刷新缓冲写入器。
// 缓冲满触发的预 flush 成功后刷新 lastFlushAt。
func (p *Poller) flushBufferedWriter(content []byte, lastFlushAt *int64, nowMilli int64) (int, error) {
	sizeOfContent := len(content)
	if sizeOfContent == 0 {
		return 0, nil
	}

	if sizeOfContent > p.writer.Available() && p.writer.Buffered() > 0 {
		if err := p.writer.Flush(); err != nil {
			return 0, err
		}
		*lastFlushAt = nowMilli
	}

	return p.writer.Write(content)
}

// CleanQueue 清理队列中的所有内容。
// 在 Stop 流程中于 Run 退出后调用，复用 drain 的批量/逐条策略。
func (p *Poller) CleanQueue() {
	nowMilli := time.Now().UnixMilli()
	executeAt := nowMilli
	lastFlushAt := nowMilli
	p.drain(&executeAt, &lastFlushAt, nowMilli)
}
