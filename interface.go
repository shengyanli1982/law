package law

import "bytes"

// Writer 定义了写入器接口
type Writer interface {
	// Write 写入数据，返回写入的字节数和可能的错误
	Write([]byte) (int, error)

	// Stop 停止写入器
	Stop()
}

// 编译期断言：*WriteAsyncer 实现 Writer 接口。
// Writer 为历史公共 API（向后兼容保留，不可删除），
// 此断言防止未来对 WriteAsyncer 的 Write/Stop 签名调整时悄然破坏该契约。
var _ Writer = (*WriteAsyncer)(nil)

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

	// Pop 从队列中取出值。
	// 契约：Pop 必须非阻塞——队列为空时立即返回 nil，禁止阻塞等待。
	// 内部 poller 的 drain 循环依赖"空队列时 Pop 立即返回 nil"判定队列已排空；
	// 若自定义队列的 Pop 在空队列上阻塞，poller 将挂死（flush 饿死、Stop 永久阻塞）。
	Pop() *bytes.Buffer
}
