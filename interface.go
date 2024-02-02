package law

import "log"

// Writer 是一个定义了写入行为的接口。
// Writer is an interface that defines the behavior of a writer.
type Writer interface {
	Write([]byte) (int, error)
	Stop()
}

// Callback 是一个定义了回调方法的接口。
// Callback is an interface that defines the callback methods.
type Callback interface {
	OnPushQueue([]byte)
	OnPopQueue([]byte, int64)
	OnWrite([]byte)
}

// emptyCallback 是一个实现了 Callback 接口的空方法的结构体。
// emptyCallback is a struct that implements the Callback interface with empty methods.
type emptyCallback struct{}

func (c *emptyCallback) OnPushQueue([]byte)       {} // empty method
func (c *emptyCallback) OnPopQueue([]byte, int64) {} // empty method
func (c *emptyCallback) OnWrite([]byte)           {} // empty method

// newEmptyCallback 是一个返回 emptyCallback 实例的函数。
// newEmptyCallback is a function that returns an instance of emptyCallback.
func newEmptyCallback() Callback {
	return &emptyCallback{}
}

// QueueInterface 是一个定义了队列行为的接口。
// QueueInterface is an interface that defines the behavior of a queue.
type QueueInterface interface {
	Push(value any)
	Pop() any
}

// Logger 是一个定义了日志行为的接口。
// Logger is an interface that defines the behavior of a logger.
type Logger interface {
	Errorf(format string, args ...any)
}

// logger 是一个实现了 Logger 接口的结构体。
// logger is a struct that implements the Logger interface.
type logger struct{}

// Errorf 是一个实现了 Logger 接口的方法。
// Errorf is a method that implements the Logger interface.
func (l *logger) Errorf(format string, args ...any) {
	log.Printf(format, args...)
}

// newLogger 是一个返回 logger 实例的函数。
// newLogger is a function that returns an instance of logger.
func newLogger() Logger {
	return &logger{}
}
