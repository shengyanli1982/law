package zapasyncwriter

import "errors"

var (
	ErrorConfigFileNotFound = errors.New("config file not found")
	ErrorQueueIsFull        = errors.New("queue is full")
	ErrorQueueIsClosed      = errors.New("queue is closed")
	ErrorWriterIsClosed     = errors.New("writer is closed")
)
