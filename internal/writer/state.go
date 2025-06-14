package writer

import "sync/atomic"

// 缓存行大小（字节）
// Cache line size (bytes)
const cacheLinePadSize = 64

// 填充大小（以uint64为单位）
// Padding size (in terms of uint64)
const paddingSize = (cacheLinePadSize - 8) / 8

// Status 结构体用于表示写异步器的状态
// The Status struct is used to represent the status of the write asyncer
type Status struct {
	// 预填充，避免前面的字段与当前结构体的字段共享缓存行
	// Pre-padding to avoid cache line sharing between previous fields and the fields of the current struct
	_padding1 [paddingSize]uint64

	// running 表示写异步器是否正在运行
	// running indicates whether the write asyncer is running
	running atomic.Bool

	// 填充，避免 running 和 executeAt 共享缓存行
	// Padding to avoid cache line sharing between running and executeAt
	_padding2 [paddingSize]uint64

	// executeAt 表示下一次执行的时间
	// executeAt represents the time of the next execution
	executeAt atomic.Int64

	// 后填充，避免后面的字段与当前结构体的字段共享缓存行
	// Post-padding to avoid cache line sharing between the fields of the current struct and subsequent fields
	_padding3 [paddingSize]uint64
}

// NewStatus 是一个函数，它创建并返回一个新的 Status
// NewStatus is a function that creates and returns a new Status
func NewStatus() *Status {
	return &Status{
		// running 是一个原子布尔值，它的初始值为 false
		// running is an atomic boolean with an initial value of false
		running: atomic.Bool{},

		// executeAt 是一个原子 int64 值，它的初始值为 0
		// executeAt is an atomic int64 with an initial value of 0
		executeAt: atomic.Int64{},
	}
}

// IsRunning 是一个方法，它返回 Status 的 running 字段的值
// IsRunning is a method that returns the value of the running field of Status
func (s *Status) IsRunning() bool {
	return s.running.Load()
}

// SetRunning 是一个方法，它设置 Status 的 running 字段的值
// SetRunning is a method that sets the value of the running field of Status
func (s *Status) SetRunning(running bool) {
	s.running.Store(running)
}

// GetExecuteAt 是一个方法，它返回 Status 的 executeAt 字段的值
// GetExecuteAt is a method that returns the value of the executeAt field of Status
func (s *Status) GetExecuteAt() int64 {
	return s.executeAt.Load()
}

// SetExecuteAt 是一个方法，它设置 Status 的 executeAt 字段的值
// SetExecuteAt is a method that sets the value of the executeAt field of Status
func (s *Status) SetExecuteAt(executeAt int64) {
	s.executeAt.Store(executeAt)
}
