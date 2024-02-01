package law

import (
	"bufio"
	"context"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shengyanli1982/law/internal/pool"
	lfq "github.com/shengyanli1982/law/internal/queue"
	"github.com/shengyanli1982/law/internal/util"
)

const (
	defaultHeartbeatInterval = 500 * time.Millisecond
	defaultIdleTimeout       = 5 * time.Second
)

type Element struct {
	buffer   []byte
	updateAt int64
}

func (e *Element) Reset() {
	e.buffer = nil
	e.updateAt = 0
}

type Status struct {
	running   atomic.Bool
	executeAt atomic.Int64
}

type WriteAsyncer struct {
	config      *Config
	queue       QueueInterface
	writer      io.Writer
	bWriter     *bufio.Writer
	timer       atomic.Int64
	once        sync.Once
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	state       Status
	elementpool *pool.Pool
}

func NewWriteAsyncer(writer io.Writer, conf *Config) *WriteAsyncer {
	if writer == nil {
		writer = os.Stdout
	}

	conf = isConfigValid(conf)

	wa := &WriteAsyncer{
		config:      conf,
		queue:       lfq.NewLockFreeQueue(),
		writer:      writer,
		bWriter:     bufio.NewWriterSize(writer, conf.buffsize),
		state:       Status{},
		elementpool: pool.NewPool(func() any { return &Element{} }, lfq.NewLockFreeQueue()),
		timer:       atomic.Int64{},
		once:        sync.Once{},
		wg:          sync.WaitGroup{},
	}
	wa.ctx, wa.cancel = context.WithCancel(context.Background())
	wa.state.executeAt.Store(time.Now().UnixMilli())
	wa.state.running.Store(true)

	wa.wg.Add(2)
	go wa.poller()
	go wa.updateTimer()

	return wa
}

func (wa *WriteAsyncer) Stop() {
	wa.once.Do(func() {
		wa.state.running.Store(false)
		wa.cancel()
		wa.wg.Wait()
		wa.bWriter.Flush()
	})
}

func (wa *WriteAsyncer) Write(p []byte) (n int, err error) {
	element := wa.elementpool.Get().(*Element)
	element.buffer = p
	element.updateAt = wa.timer.Load()
	wa.queue.Push(element)
	wa.config.callback.OnPushQueue(p)
	return len(p), nil
}

func (wa *WriteAsyncer) bufferedWriter(p []byte) (int, error) {
	wa.config.callback.OnWrite(p)
	if len(p) > wa.bWriter.Available() && wa.bWriter.Buffered() > 0 {
		if err := wa.bWriter.Flush(); err != nil {
			return wa.writer.Write(p)
		}
	}
	return wa.bWriter.Write(p)
}

func (wa *WriteAsyncer) poller() {
	heartbeat := time.NewTicker(defaultHeartbeatInterval)

	defer func() {
		heartbeat.Stop()
		wa.wg.Done()
	}()

	executeFunc := func(element *Element) {
		now := wa.timer.Load()
		wa.state.executeAt.Store(now)
		wa.config.callback.OnPopQueue(element.buffer, now-element.updateAt)
		if _, err := wa.bufferedWriter(element.buffer); err != nil {
			wa.config.logger.Errorf("data write error, error: %s, message: %s", err.Error(), util.BytesToString(element.buffer))
		}
		element.Reset()
		wa.elementpool.Put(element)
	}

	for {
		select {
		case <-wa.ctx.Done():
			for {
				element := wa.queue.Pop()
				if element == nil {
					break
				}
				executeFunc(element.(*Element))
			}
			return
		default:
			element := wa.queue.Pop()
			if element != nil {
				executeFunc(element.(*Element))
			} else {
				<-heartbeat.C
				now := wa.timer.Load()
				if wa.bWriter.Buffered() > 0 && now-wa.state.executeAt.Load() > defaultIdleTimeout.Milliseconds() {
					if err := wa.bWriter.Flush(); err != nil {
						wa.config.logger.Errorf("buffered writer flush error, error: %s", err.Error())
					}
					wa.state.executeAt.Store(now)
				}
			}
		}
	}
}

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
