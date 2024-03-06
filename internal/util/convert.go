package util

import "unsafe"

// BytesToString 将字节切片转换为字符串，不进行内存拷贝
// BytesToString converts a slice of bytes to a string without memory copy
func BytesToString(b []byte) string {
	// 使用unsafe.Pointer进行类型转换，避免内存拷贝
	// Use unsafe.Pointer for type conversion to avoid memory copy
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytes 将字符串转换为字节切片，不进行内存拷贝
// StringToBytes converts a string to a slice of bytes without memory copy
func StringToBytes(s string) []byte {
	// 获取字符串的起始地址和长度
	// Get the start address and length of the string
	x := (*[2]uintptr)(unsafe.Pointer(&s))

	// 创建一个新的切片头部，包含起始地址、长度和容量
	// Create a new slice header, including start address, length, and capacity
	h := [3]uintptr{x[0], x[1], x[1]}

	// 使用unsafe.Pointer进行类型转换，避免内存拷贝
	// Use unsafe.Pointer for type conversion to avoid memory copy
	return *(*[]byte)(unsafe.Pointer(&h))
}
