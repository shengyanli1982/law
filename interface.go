package law

import "log"

type Writer interface {
	Write([]byte) (int, error)
	Stop()
}

type Callback interface {
	OnPushQueue([]byte)
	OnPopQueue([]byte, int64)
	OnWrite([]byte)
}

type emptyCallback struct{}

func (c *emptyCallback) OnPushQueue([]byte)       {}
func (c *emptyCallback) OnPopQueue([]byte, int64) {}
func (c *emptyCallback) OnWrite([]byte)           {}

func newEmptyCallback() *emptyCallback {
	return &emptyCallback{}
}

type QueueInterface interface {
	Push(value any)
	Pop() any
}

type Logger interface {
	Errorf(format string, args ...any)
}

type logger struct{}

func (l *logger) Errorf(format string, args ...any) {
	log.Printf(format, args...)
}

func newLogger() *logger {
	return &logger{}
}
