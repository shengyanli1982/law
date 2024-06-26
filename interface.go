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
	// OnWriteFailed 是一个方法，当写操作失败时会被调用。
	// 它接受两个参数：一个字节切片（表示写入内容）和一个错误（表示失败的原因）。
	// OnWriteFailed is a method that is called when a write operation fails.
	// It takes two parameters: a byte slice (indicating the content to be written) and an error (indicating the reason for the failure).
	OnWriteFailed(content []byte, reason error)
}

// emptyCallback 是一个实现了 Callback 接口的结构体，但所有方法的实现都为空。
// emptyCallback is a struct that implements the Callback interface, but all method implementations are empty.
type emptyCallback struct{}

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

// Queue 是一个接口，定义了队列的基本操作：Push 和 Pop。
// Queue is an interface that defines the basic operations of a queue: Push and Pop.
type Queue interface {
	// Push 方法用于将值添加到队列中。
	// The Push method is used to add a value to the queue.
	Push(value interface{})

	// Pop 方法用于从队列中取出一个值。
	// The Pop method is used to take a value out of the queue.
	Pop() interface{}
}
