package law

// Writer 定义了写入器接口
// Writer defines the writer interface
type Writer interface {
	// Write 写入数据，返回写入的字节数和可能的错误
	// Write writes data and returns the number of bytes written and any error
	Write([]byte) (int, error)

	// Stop 停止写入器
	// Stop stops the writer
	Stop()
}

// Callback 定义了回调接口
// Callback defines the callback interface
type Callback interface {
	// OnWriteFailed 当写入失败时被调用
	// OnWriteFailed is called when writing fails
	OnWriteFailed(content []byte, reason error)
}

// emptyCallback 空回调实现
// emptyCallback is an empty callback implementation
type emptyCallback struct{}

// OnWriteFailed 空回调的写入失败处理方法（无操作）
// OnWriteFailed handles write failures for empty callback (no-op)
func (c *emptyCallback) OnWriteFailed([]byte, error) {}

// newEmptyCallback 创建新的空回调实例
// newEmptyCallback creates a new empty callback instance
func newEmptyCallback() Callback {
	return &emptyCallback{}
}

// Queue 定义了队列接口
// Queue defines the queue interface
type Queue interface {
	// Push 将值推入队列
	// Push pushes a value into the queue
	Push(value interface{})

	// Pop 从队列中取出值
	// Pop retrieves a value from the queue
	Pop() interface{}
}
