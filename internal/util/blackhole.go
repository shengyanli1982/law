package util

// BlackHoleWriter 是一个不做任何操作的Writer
// BlackHoleWriter is a Writer that does nothing
type BlackHoleWriter struct{}

// Write 实现了io.Writer接口，但不做任何操作，只返回输入数据的长度和nil错误
// Write implements the io.Writer interface, but does nothing, 
// it only returns the length of the input data and a nil error
func (w *BlackHoleWriter) Write(p []byte) (int, error) {
	// 返回输入数据的长度和nil错误
	// Return the length of the input data and a nil error
	return len(p), nil
}