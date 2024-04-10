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
	defaultHeartbeatInterval = 500 * time.Millisecond

	defaultIdleTimeout = 5 * time.Second
)

var ErrorWriteAsyncerIsClosed = errors.New("write asyncer is closed")

type Element struct {
	buffer []byte

	updateAt int64
}

func (e *Element) Reset() {

	e.buffer = nil

	e.updateAt = 0

}

type Status struct {
	running atomic.Bool

	executeAt atomic.Int64
}

type WriteAsyncer struct {
	config *Config

	queue QueueInterface

	writer io.Writer

	bufferedWriter *bufio.Writer

	timer atomic.Int64

	once sync.Once

	ctx context.Context

	cancel context.CancelFunc

	wg sync.WaitGroup

	state Status

	elementpool *pool.Pool
}

func NewWriteAsyncer(writer io.Writer, conf *Config) *WriteAsyncer {

	if writer == nil {

		writer = os.Stdout

	}

	conf = isConfigValid(conf)

	wa := &WriteAsyncer{

		config: conf,

		queue: lfq.NewLockFreeQueue(),

		writer: writer,

		bufferedWriter: bufio.NewWriterSize(writer, conf.buffsize),

		state: Status{},

		elementpool: pool.NewPool(func() any { return &Element{} }, lfs.NewLockFreeStack()),

		timer: atomic.Int64{},

		once: sync.Once{},

		wg: sync.WaitGroup{},
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

		wa.cleaningQueueToWriter()

		wa.bufferedWriter.Flush()

	})

}

func (wa *WriteAsyncer) Write(p []byte) (n int, err error) {

	if !wa.state.running.Load() {

		return 0, ErrorWriteAsyncerIsClosed

	}

	element := wa.elementpool.Get().(*Element)

	element.buffer = p

	element.updateAt = wa.timer.Load()

	wa.queue.Push(element)

	wa.config.callback.OnPushQueue(p)

	return len(p), nil

}

func (wa *WriteAsyncer) flushBufferedWriter(p []byte) (int, error) {

	wa.config.callback.OnWrite(p)

	if len(p) > wa.bufferedWriter.Available() && wa.bufferedWriter.Buffered() > 0 {

		if err := wa.bufferedWriter.Flush(); err != nil {

			return wa.writer.Write(p)

		}

	}

	return wa.bufferedWriter.Write(p)

}

func (wa *WriteAsyncer) poller() {

	heartbeat := time.NewTicker(defaultHeartbeatInterval)

	defer func() {

		heartbeat.Stop()

		wa.wg.Done()

	}()

	for {

		elem := wa.queue.Pop()

		if elem != nil {

			wa.executeFunc(elem.(*Element))

		} else {

			select {

			case <-wa.ctx.Done():

				return

			case <-heartbeat.C:

				now := wa.timer.Load()

				diff := now - wa.state.executeAt.Load()

				if wa.bufferedWriter.Buffered() > 0 && diff >= defaultIdleTimeout.Milliseconds() {

					if err := wa.bufferedWriter.Flush(); err != nil {

						wa.config.logger.Errorf("buffered writer flush error, error: %s", err.Error())

					}

					wa.state.executeAt.Store(now)

				}

				if diff > defaultIdleTimeout.Milliseconds()*6 {

					wa.elementpool.Prune()

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

func (wa *WriteAsyncer) executeFunc(elem *Element) {

	now := wa.timer.Load()

	wa.state.executeAt.Store(now)

	wa.config.callback.OnPopQueue(elem.buffer, now-elem.updateAt)

	if _, err := wa.flushBufferedWriter(elem.buffer); err != nil {

		wa.config.logger.Errorf("data write error, error: %s, message: %s", err.Error(), util.BytesToString(elem.buffer))

	}

	elem.Reset()

	wa.elementpool.Put(elem)

}

func (wa *WriteAsyncer) cleaningQueueToWriter() {

	for {

		elem := wa.queue.Pop()

		if elem == nil {

			break

		}

		wa.executeFunc(elem.(*Element))

	}

}
