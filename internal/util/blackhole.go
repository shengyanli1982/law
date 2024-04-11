package util

// BlackHoleWriter 是一个结构体，实现了 io.Writer 接口，但实际上并不执行任何写入操作。
// BlackHoleWriter is a struct that implements the io.Writer interface, but actually does not perform any write operations.
type BlackHoleWriter struct{}

// Write 是 BlackHoleWriter 结构体实现 io.Writer 接口的方法，它接收一个字节切片 p，但并不执行实际的写入操作。
// Write is a method of the BlackHoleWriter struct that implements the io.Writer interface. It receives a byte slice p, but does not perform the actual write operation.
func (w *BlackHoleWriter) Write(p []byte) (int, error) {
	// 直接返回 p 的长度和 nil 错误，表示所有的字节都已经被 "写入"，实际上并没有写入任何地方。
	// Directly return the length of p and a nil error, indicating that all bytes have been "written", but actually nothing has been written anywhere.
	return len(p), nil
}
