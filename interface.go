package law

import "log"

// Writer 是一个定义了写入行为的接口。
// Writer is an interface that defines the behavior of a writer.
type Writer interface {
	// Write 方法接受一个字节切片，尝试写入数据，并返回写入的字节数和可能的错误。
	// The Write method takes a slice of bytes, attempts to write data, and returns the number of bytes written and any possible error.
	Write([]byte) (int, error)

	// Stop 方法停止写入操作。
	// The Stop method stops the writing operation.
	Stop()
}

// Callback 是一个定义了回调方法的接口。
// Callback is an interface that defines the callback methods.
type Callback interface {
	// OnPushQueue 方法在数据被推入队列时被调用。
	// The OnPushQueue method is called when data is pushed into the queue.
	OnPushQueue([]byte)

	// OnPopQueue 方法在数据从队列中弹出时被调用。
	// The OnPopQueue method is called when data is popped from the queue.
	OnPopQueue([]byte, int64)

	// OnWrite 方法在数据被写入时被调用。
	// The OnWrite method is called when data is written.
	OnWrite([]byte)
}

// emptyCallback 是一个实现了 Callback 接口的空方法的结构体。
// emptyCallback is a struct that implements the Callback interface with empty methods.
type emptyCallback struct{}

// OnPushQueue 是一个空方法，它在数据被推入队列时被调用。
// OnPushQueue is an empty method that is called when data is pushed into the queue.
func (c *emptyCallback) OnPushQueue([]byte) {}

// OnPopQueue 是一个空方法，它在数据从队列中弹出时被调用。
// OnPopQueue is an empty method that is called when data is popped from the queue.
func (c *emptyCallback) OnPopQueue([]byte, int64) {}

// OnWrite 是一个空方法，它在数据被写入时被调用。
// OnWrite is an empty method that is called when data is written.
func (c *emptyCallback) OnWrite([]byte) {}

// newEmptyCallback 是一个返回 emptyCallback 实例的函数。
// newEmptyCallback is a function that returns an instance of emptyCallback.
func newEmptyCallback() Callback {
	return &emptyCallback{}
}

// QueueInterface 是一个定义了队列行为的接口。
// QueueInterface is an interface that defines the behavior of a queue.
type QueueInterface interface {
	// Push 方法接受一个值，尝试将其推入队列。
	// The Push method takes a value and attempts to push it into the queue.
	Push(value any)

	// Pop 方法尝试从队列中弹出一个值。
	// The Pop method attempts to pop a value from the queue.
	Pop() any
}

// Logger 是一个定义了日志行为的接口。
// Logger is an interface that defines the behavior of a logger.
type Logger interface {
	// Errorf 方法接受一个格式化字符串和一些参数，然后记录一条错误日志。
	// The Errorf method takes a format string and some arguments, then logs an error message.
	Errorf(format string, args ...any)
}

// logger 是一个实现了 Logger 接口的结构体。
// logger is a struct that implements the Logger interface.
type logger struct{}

// Errorf 是一个实现了 Logger 接口的方法。
// Errorf is a method that implements the Logger interface.
func (l *logger) Errorf(format string, args ...any) {
	// 使用 log 包的 Printf 函数记录一条错误日志。
	// Use the Printf function of the log package to log an error message.
	log.Printf(format, args...)
}

// newLogger 是一个返回 logger 实例的函数。
// newLogger is a function that returns an instance of logger.
func newLogger() Logger {
	return &logger{}
}
