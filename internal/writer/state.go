package writer

import (
	"sync/atomic"
)

// Status 结构体用于表示写异步器的运行状态
type Status struct {
	running atomic.Bool
}

// NewStatus 是一个函数，它创建并返回一个新的 Status
func NewStatus() *Status {
	return &Status{
		running: atomic.Bool{},
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
