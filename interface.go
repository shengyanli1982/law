package law

// Writer 定义了写入器接口
type Writer interface {
	// Write 写入数据，返回写入的字节数和可能的错误
	Write([]byte) (int, error)

	// Stop 停止写入器
	Stop()
}

// Callback 定义了回调接口
type Callback interface {
	// OnWriteFailed 当写入失败时被调用
	OnWriteFailed(content []byte, reason error)
}

// emptyCallback 空回调实现
type emptyCallback struct{}

// OnWriteFailed 空回调的写入失败处理方法（无操作）
func (c *emptyCallback) OnWriteFailed([]byte, error) {}

// defaultEmptyCallback 包级私有单例
var defaultEmptyCallback = &emptyCallback{}

// newEmptyCallback 返回空回调单例
func newEmptyCallback() Callback {
	return defaultEmptyCallback
}

// Queue 定义了队列接口
type Queue interface {
	// Push 将值推入队列
	Push(value interface{})

	// Pop 从队列中取出值
	Pop() interface{}
}
