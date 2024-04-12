package law

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

	// OnWriteSuccess 方法在数据成功写入时调用。
	// The OnWriteSuccess method is called when data is successfully written.
	OnWriteSuccess([]byte)

	// OnWriteFailed 方法在数据写入失败时调用，会传入失败的错误信息。
	// The OnWriteFailed method is called when data writing fails, and the failure error information will be passed in.
	OnWriteFailed([]byte, error)
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

// OnWriteSuccess 是 emptyCallback 结构体实现 Callback 接口的方法，但此方法没有任何实现。
// OnWriteSuccess is a method of the emptyCallback struct that implements the Callback interface, but this method has no implementation.
func (c *emptyCallback) OnWriteSuccess([]byte) {}

// OnWriteFailed 是 emptyCallback 结构体实现 Callback 接口的方法，但此方法没有任何实现。
// OnWriteFailed is a method of the emptyCallback struct that implements the Callback interface, but this method has no implementation.
func (c *emptyCallback) OnWriteFailed([]byte, error) {}

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
