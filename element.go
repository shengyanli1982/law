package law

// Element 是一个结构体，包含一个字节缓冲区和一个更新时间戳
// Element is a structure that contains a byte buffer and an update timestamp
type Element struct {
	// buffer 是一个字节缓冲区
	// buffer is a byte buffer
	buffer []byte

	// updateAt 是一个更新时间戳
	// updateAt is an update timestamp
	updateAt int64
}

// NewElement 是一个构造函数，用于创建一个新的 Element 实例
// NewElement is a constructor function for creating a new Element instance
func NewElement() *Element {
	// 返回一个新的 Element 实例
	// Return a new Element instance
	return &Element{}

}

// Reset 是 Element 结构体的一个方法，用于重置 Element 的状态
// Reset is a method of the Element structure, used to reset the state of the Element
func (e *Element) Reset() {
	// 清空字节缓冲区
	// Empty the byte buffer
	e.buffer = nil

	// 重置更新时间戳
	// Reset the update timestamp
	e.updateAt = 0

}
