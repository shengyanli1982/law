package writer

import "sync/atomic"

// Status 结构体用于表示写异步器的状态
// The Status struct is used to represent the status of the write asyncer
type Status struct {
	// running 表示写异步器是否正在运行
	// running indicates whether the write asyncer is running
	running atomic.Bool

	// executeAt 表示下一次执行的时间
	// executeAt represents the time of the next execution
	executeAt atomic.Int64
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
