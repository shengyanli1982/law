package law

import (
	"context"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	lfq "github.com/shengyanli1982/law/internal/queue"
)

type Element struct {
	buff     []byte
	updateAt int64
}

type WriteAsyncer struct {
	config *Config
	queue  QueueInterface
	writer io.Writer
	timer  atomic.Int64
	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewWriteAsyncer(w io.Writer, queue QueueInterface, conf *Config) *WriteAsyncer {
	if w == nil {
		w = os.Stdout
	}
	if queue == nil {
		queue = lfq.NewLockFreeQueue()
	}

	wa := WriteAsyncer{
		config: isConfigValid(conf),
		queue:  queue,
		writer: w,
		timer:  atomic.Int64{},
		once:   sync.Once{},
		wg:     sync.WaitGroup{},
	}
	wa.ctx, wa.cancel = context.WithCancel(context.Background())

	wa.wg.Add(1)
	go wa.poller()

	// 启动时间定时器
	// start time timer.
	wa.wg.Add(1)
	go func() {
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
	}()

	return &wa
}

func (wa *WriteAsyncer) Stop() {
	wa.once.Do(func() {
		wa.cancel()
		wa.wg.Wait()
	})
}

func (wa *WriteAsyncer) Write(p []byte) (n int, err error) {
	wa.queue.Push(&Element{
		buff:     p,
		updateAt: wa.timer.Load(),
	})
	return len(p), nil
}

func (wa *WriteAsyncer) poller() {
	heartbeat := time.NewTicker(time.Millisecond * 500) // 心跳 (Heartbeat)

	defer func() {
		heartbeat.Stop()
		wa.wg.Done()
	}()

	executeFunc := func(element *Element) {

	}

	for {
		select {
		case <-wa.ctx.Done(): // 如果停止上下文被取消，则返回 (If the stop context is canceled, return)
			return
		default:
			element := wa.queue.Pop() // 从队列中取出链表节点 (Pop list node from the queue)
			if element != nil {       // 如果链表节点不为空，则执行 (If the list node is not empty, execute)
				executeFunc(element.(*Element)) // 执行 (Execute)
			} else {
				<-heartbeat.C // 如果队列为空，则等待心跳 (If the queue is empty, wait for heartbeat)
			}
		}
	}
}
