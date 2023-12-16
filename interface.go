package zapasyncwriter

// Writer is the common interface for all log writers.
// Writer 是所有日志写入器的通用接口。
type Writer interface {
	Write([]byte) (int, error) // Write implements io.Writer
	Stop()                     // Stop stops the writer flushing any buffered log entries.
}

// Callback is the interface which do action push/pop/write will call.
// Callback 是执行 push/pop/write 操作时调用的接口。
type Callback interface {
	OnPushQueue([]byte)       // OnPushQueue is called when a log entry is pushed into the queue.
	OnPopQueue([]byte, int64) // OnPopQueue is called when a log entry is popped from the queue.
	OnWrite([]byte)           // OnWrite is called when a log entry is written to the underlying writer.
}

// emptyCallback is the default implementation of Callback interface.
// emptyCallback 是 Callback 接口的默认实现。
type emptyCallback struct{}

func (c *emptyCallback) OnPushQueue([]byte)       {} // do nothing
func (c *emptyCallback) OnPopQueue([]byte, int64) {} // do nothing
func (c *emptyCallback) OnWrite([]byte)           {} // do nothing
