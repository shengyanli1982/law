package law

import "log"

// Writer 是一个接口，定义了写操作的行为。
// Writer is an interface that defines the behavior of write operations.
type Writer interface {
	// Write 方法接受一个字节切片，返回写入的字节数和可能的错误。
	// The Write method accepts a byte slice and returns the number of bytes written and a possible error.
	Write([]byte) (int, error)

	// Stop 方法用于停止写操作。
	// The Stop method is used to stop write operations.
	Stop()
}

// Callback 是一个接口，定义了队列操作和写操作的回调函数。
// Callback is an interface that defines callback functions for queue operations and write operations.
type Callback interface {
	// OnPushQueue 方法在数据被推入队列时调用。
	// The OnPushQueue method is called when data is pushed into the queue.
	OnPushQueue([]byte)

	// OnPopQueue 方法在数据从队列中弹出时调用。
	// The OnPopQueue method is called when data is popped from the queue.
	OnPopQueue([]byte, int64)

	// OnWrite 方法在数据被写入时调用。
	// The OnWrite method is called when data is written.
	OnWrite([]byte)
}

// emptyCallback 是一个实现了 Callback 接口的结构体，但所有方法的实现都为空。
// emptyCallback is a struct that implements the Callback interface, but all method implementations are empty.
type emptyCallback struct{}

// OnPushQueue 是 emptyCallback 结构体实现 Callback 接口的方法，但此方法没有任何实现。
// OnPushQueue is a method of the emptyCallback struct that implements the Callback interface, but this method has no implementation.
func (c *emptyCallback) OnPushQueue([]byte) {}

// OnPopQueue 是 emptyCallback 结构体实现 Callback 接口的方法，但此方法没有任何实现。
// OnPopQueue is a method of the emptyCallback struct that implements the Callback interface, but this method has no implementation.
func (c *emptyCallback) OnPopQueue([]byte, int64) {}

// OnWrite 是 emptyCallback 结构体实现 Callback 接口的方法，但此方法没有任何实现。
// OnWrite is a method of the emptyCallback struct that implements the Callback interface, but this method has no implementation.
func (c *emptyCallback) OnWrite([]byte) {}

// newEmptyCallback 是一个构造函数，用于创建一个新的 emptyCallback 实例。
// newEmptyCallback is a constructor function for creating a new emptyCallback instance.
func newEmptyCallback() Callback {
	// 返回一个新的 emptyCallback 实例。
	// Return a new emptyCallback instance.
	return &emptyCallback{}
}

// QueueInterface 是一个接口，定义了队列的基本操作：Push 和 Pop。
// QueueInterface is an interface that defines the basic operations of a queue: Push and Pop.
type QueueInterface interface {
	// Push 方法用于将值添加到队列中。
	// The Push method is used to add a value to the queue.
	Push(value interface{})

	// Pop 方法用于从队列中取出一个值。
	// The Pop method is used to take a value out of the queue.
	Pop() interface{}
}

// Logger 是一个接口，定义了日志记录的基本操作：Errorf。
// Logger is an interface that defines the basic operations of logging: Errorf.
type Logger interface {
	// Errorf 方法用于记录错误信息。
	// The Errorf method is used to log error information.
	Errorf(format string, args ...interface{})
}

// logger 是 Logger 接口的一个实现。
// logger is an implementation of the Logger interface.
type logger struct{}

// Errorf 是 logger 结构体实现 Logger 接口的方法，用于记录错误信息。
// Errorf is a method of the logger struct that implements the Logger interface, used to log error information.
func (l *logger) Errorf(format string, args ...interface{}) {
	// 使用 log 包的 Printf 函数来记录错误信息。
	// Use the Printf function of the log package to log error information.
	log.Printf(format, args...)
}

// newLogger 是一个构造函数，用于创建一个新的 logger 实例。
// newLogger is a constructor function for creating a new logger instance.
func newLogger() Logger {
	// 返回一个新的 logger 实例。
	// Return a new logger instance.
	return &logger{}
}
