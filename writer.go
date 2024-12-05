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

const defaultHeartbeatInterval = 500 * time.Millisecond
const defaultIdleTimeout = 5 * time.Second

var (
	ErrorWriteAsyncerIsClosed = errors.New("write asyncer is closed")
	ErrorWriteContentIsNil    = errors.New("write content is nil")
)

type WriteAsyncer struct {
	config         *Config
	writer         io.Writer
	bufferedWriter *bufio.Writer
	timer          atomic.Int64
	once           sync.Once
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	state          *wr.Status
	bufferpool     *wr.BufferPool
}

func NewWriteAsyncer(writer io.Writer, conf *Config) *WriteAsyncer {
	if writer == nil {
		writer = os.Stdout
	}

	conf = isConfigValid(conf)

	wa := &WriteAsyncer{
		config:         conf,
		writer:         writer,
		bufferedWriter: bufio.NewWriterSize(writer, conf.buffSize),
		state:          wr.NewStatus(),
		timer:          atomic.Int64{},
		once:           sync.Once{},
		wg:             sync.WaitGroup{},
		bufferpool:     wr.NewBufferPool(),
	}

	wa.ctx, wa.cancel = context.WithCancel(context.Background())
	wa.state.SetExecuteAt(time.Now().UnixMilli())
	wa.state.SetRunning(true)

	wa.wg.Add(2)
	go wa.poller()
	go wa.updateTimer()

	return wa
}

func (wa *WriteAsyncer) Stop() {
	wa.once.Do(func() {
		wa.state.SetRunning(false)
		wa.cancel()
		wa.wg.Wait()
		wa.cleanQueueToWriter()
		wa.bufferedWriter.Flush()
		wa.bufferedWriter.Reset(io.Discard)
	})
}

func (wa *WriteAsyncer) Write(p []byte) (n int, err error) {
	if !wa.state.IsRunning() {
		return 0, ErrorWriteAsyncerIsClosed
	}

	if p == nil {
		return 0, ErrorWriteContentIsNil
	}

	l := len(p)
	if l <= 0 {
		return 0, nil
	}

	buff := wa.bufferpool.Get()
	buff.Grow(l)

	if n, err = buff.Write(p); err != nil {
		wa.bufferpool.Put(buff)
		return 0, err
	}

	wa.config.queue.Push(buff)
	return l, nil
}

func (wa *WriteAsyncer) flushBufferedWriter(content []byte) (int, error) {
	sizeOfContent := len(content)
	if sizeOfContent == 0 {
		return 0, nil
	}

	if sizeOfContent > wa.bufferedWriter.Available() && wa.bufferedWriter.Buffered() > 0 {
		if err := wa.bufferedWriter.Flush(); err != nil {
			return 0, err
		}
	}

	if sizeOfContent >= wa.config.buffSize {
		return wa.writer.Write(content)
	}

	return wa.bufferedWriter.Write(content)
}

func (wa *WriteAsyncer) poller() {
	const checkInterval = defaultHeartbeatInterval
	heartbeat := time.NewTicker(checkInterval)

	defer func() {
		heartbeat.Stop()
		wa.wg.Done()
	}()

	for {
		if element := wa.config.queue.Pop(); element != nil {
			wa.executeFunc(element.(*bytes.Buffer))
			continue
		}

		select {
		case <-wa.ctx.Done():
			return

		case <-heartbeat.C:
			if wa.bufferedWriter.Buffered() > 0 {
				now := wa.timer.Load()
				if (now - wa.state.GetExecuteAt()) >= defaultIdleTimeout.Milliseconds() {
					if err := wa.bufferedWriter.Flush(); err != nil {
						wa.config.callback.OnWriteFailed(nil, err)
					}
					wa.state.SetExecuteAt(now)
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

func (wa *WriteAsyncer) executeFunc(buff *bytes.Buffer) {
	wa.state.SetExecuteAt(wa.timer.Load())
	content := buff.Bytes()

	if _, err := wa.flushBufferedWriter(content); err != nil {
		failContent := make([]byte, len(content))
		copy(failContent, content)
		wa.config.callback.OnWriteFailed(failContent, err)
	}

	wa.bufferpool.Put(buff)
}

func (wa *WriteAsyncer) cleanQueueToWriter() {
	for {
		elem := wa.config.queue.Pop()
		if elem == nil {
			break
		}
		wa.executeFunc(elem.(*bytes.Buffer))
	}
}
