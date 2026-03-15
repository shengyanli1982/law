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

// Queue 定义了内部轮询器使用的类型化队列接口。
type Queue[T any] interface {
	Push(value T)
	Pop() T
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
	executeAt         int64
	bufferpool        *wr.BufferPool
	timer             *atomic.Int64
	heartbeatInterval time.Duration
	idleTimeout       time.Duration
}

// Config Poller配置。
type Config struct {
	Queue             Queue[*bytes.Buffer]
	Writer            *bufio.Writer
	Callback          Callback
	BufferPool        *wr.BufferPool
	Timer             *atomic.Int64
	HeartbeatInterval time.Duration
	IdleTimeout       time.Duration
}

// NewPoller 创建新的轮询器。
func NewPoller(cfg *Config) *Poller {
	return &Poller{
		queue:             cfg.Queue,
		writer:            cfg.Writer,
		callback:          cfg.Callback,
		hasCallback:       cfg.Callback != nil,
		bufferpool:        cfg.BufferPool,
		timer:             cfg.Timer,
		heartbeatInterval: cfg.HeartbeatInterval,
		idleTimeout:       cfg.IdleTimeout,
	}
}

// Run 启动轮询器，处理写入请求和心跳检查。
func (p *Poller) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(p.heartbeatInterval)
	var tickCount int64

	now := time.Now().UnixMilli()
	p.timer.Store(now)
	p.executeAt = now

	defer func() {
		ticker.Stop()
		wg.Done()
	}()

	for {
		for {
			element := p.queue.Pop()
			if element == nil {
				break
			}
			p.executeFunc(element)
		}

		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			tickCount++

			if tickCount%(int64(time.Second/p.heartbeatInterval)) == 0 {
				now = time.Now().UnixMilli()
				p.timer.Store(now)
			}

			if p.writer.Buffered() > 0 {
				cachedNow := p.timer.Load()
				if (cachedNow - p.executeAt) >= p.idleTimeout.Milliseconds() {
					if err := p.writer.Flush(); err != nil {
						if p.hasCallback {
							p.callback.OnWriteFailed(nil, err)
						}
					}
					p.executeAt = cachedNow
				}
			}
		}
	}
}

// executeFunc 执行写入操作。
func (p *Poller) executeFunc(buff *bytes.Buffer) {
	p.executeAt = p.timer.Load()
	content := buff.Bytes()

	if _, err := p.flushBufferedWriter(content); err != nil {
		if p.hasCallback {
			p.callback.OnWriteFailed(content, err)
		}
	}

	p.bufferpool.Put(buff)
}

// flushBufferedWriter 刷新缓冲写入器。
func (p *Poller) flushBufferedWriter(content []byte) (int, error) {
	sizeOfContent := len(content)
	if sizeOfContent == 0 {
		return 0, nil
	}

	if sizeOfContent > p.writer.Available() && p.writer.Buffered() > 0 {
		if err := p.writer.Flush(); err != nil {
			return 0, err
		}
	}

	return p.writer.Write(content)
}

// CleanQueue 清理队列中的所有内容。
func (p *Poller) CleanQueue() {
	for {
		elem := p.queue.Pop()
		if elem == nil {
			break
		}
		p.executeFunc(elem)
	}
}
