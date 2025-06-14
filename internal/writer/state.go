package writer

import "sync/atomic"

// 缓存行大小（字节）
const cacheLinePadSize = 64

// 填充大小（以uint64为单位）
const paddingSize = (cacheLinePadSize - 8) / 8

// Status 结构体用于表示写异步器的状态
type Status struct {
	// 预填充，避免前面的字段与当前结构体的字段共享缓存行
	_padding1 [paddingSize]uint64

	// running 表示写异步器是否正在运行
	running atomic.Bool

	// 填充，避免 running 和 executeAt 共享缓存行
	_padding2 [paddingSize]uint64

	// executeAt 表示下一次执行的时间
	executeAt atomic.Int64

	// 后填充，避免后面的字段与当前结构体的字段共享缓存行
	_padding3 [paddingSize]uint64
}

// NewStatus 是一个函数，它创建并返回一个新的 Status
func NewStatus() *Status {
	return &Status{
		running: atomic.Bool{},
		executeAt: atomic.Int64{},
	}
}

// IsRunning 是一个方法，它返回 Status 的 running 字段的值
func (s *Status) IsRunning() bool {
	return s.running.Load()
}

// SetRunning 是一个方法，它设置 Status 的 running 字段的值
func (s *Status) SetRunning(running bool) {
	s.running.Store(running)
}

// GetExecuteAt 是一个方法，它返回 Status 的 executeAt 字段的值
func (s *Status) GetExecuteAt() int64 {
	return s.executeAt.Load()
}

// SetExecuteAt 是一个方法，它设置 Status 的 executeAt 字段的值
func (s *Status) SetExecuteAt(executeAt int64) {
	s.executeAt.Store(executeAt)
}
