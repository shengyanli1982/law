package law

import "bytes"

// Writer 定义了写入器接口
type Writer interface {
	// Write 写入数据，返回写入的字节数和可能的错误
	Write([]byte) (int, error)

	// Stop 停止写入器
	Stop()
}

// Callback 定义了回调接口
type Callback interface {
	// OnWriteFailed 当写入失败时被调用。
	// 注意：content 仅在回调执行期间有效，不应在回调返回后保持引用。
	// 如需异步处理 content，请在回调内复制数据。
	// content 语义：逐条写入失败时携带该条数据；
	// 预 flush 失败路径（单条消息超过 bufio 剩余空间触发的先行 flush 失败）
	// 亦属逐条失败：content 为当前条数据，且此前 bufio 缓冲中的存量数据一并丢失；
	// 批量 flush（心跳/Stop 最终 flush）失败时为 nil，
	// 表示缓冲中的批量数据丢失且不可恢复。
	OnWriteFailed(content []byte, reason error)

	// OnWriteBlocked 在有界队列已满、Push 即将阻塞时被调用。
	// reason 描述阻塞原因，例如 "bounded queue full, push will block"。
	// 用户可在此回调中实施降级策略（如丢弃低优先级日志、告警等）。
	OnWriteBlocked(reason string)
}

// emptyCallback 空回调实现
type emptyCallback struct{}

// OnWriteFailed 空回调的写入失败处理方法（无操作）
func (c *emptyCallback) OnWriteFailed([]byte, error) {}

// OnWriteBlocked 空回调的写入阻塞处理方法（无操作）
func (c *emptyCallback) OnWriteBlocked(string) {}

// defaultEmptyCallback 包级私有单例
var defaultEmptyCallback = &emptyCallback{}

// newEmptyCallback 返回空回调单例
func newEmptyCallback() Callback {
	return defaultEmptyCallback
}

// Queue 定义了队列接口
type Queue interface {
	// Push 将值推入队列，Push(nil) 行为未定义
	Push(value *bytes.Buffer)

	// Pop 从队列中取出值
	Pop() *bytes.Buffer
}
