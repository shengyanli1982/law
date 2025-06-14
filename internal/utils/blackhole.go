package utils

// BlackHoleWriter 是一个结构体，实现了 io.Writer 接口，但实际上并不执行任何写入操作。
type BlackHoleWriter struct{}

// Write 是 BlackHoleWriter 结构体实现 io.Writer 接口的方法，它接收一个字节切片 p，但并不执行实际的写入操作。
func (w *BlackHoleWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
