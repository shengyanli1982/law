package utils

import "unsafe"

// BytesToString 是一个函数，它将字节切片转换为字符串，而不进行任何内存分配。
// BytesToString is a function that converts a byte slice to a string without any memory allocation.
func BytesToString(b []byte) string {
	// 使用 unsafe 包的 Pointer 函数将字节切片的地址转换为字符串的指针，然后解引用该指针得到字符串。
	// Use the Pointer function of the unsafe package to convert the address of the byte slice to a pointer to a string, and then dereference that pointer to get the string.
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytes 是一个函数，它将字符串转换为字节切片，而不进行任何内存分配。
// StringToBytes is a function that converts a string to a byte slice without any memory allocation.
func StringToBytes(s string) []byte {
	// 使用 unsafe 包的 Pointer 函数将字符串的地址转换为 uintptr 数组的指针。
	// Use the Pointer function of the unsafe package to convert the address of the string to a pointer to a uintptr array.
	x := (*[2]uintptr)(unsafe.Pointer(&s))

	// 创建一个新的 uintptr 数组，其中包含字符串的起始地址和长度。
	// Create a new uintptr array that contains the start address and length of the string.
	h := [3]uintptr{x[0], x[1], x[1]}

	// 将 uintptr 数组的地址转换为字节切片的指针，然后解引用该指针得到字节切片。
	// Convert the address of the uintptr array to a pointer to a byte slice, and then dereference that pointer to get the byte slice.
	return *(*[]byte)(unsafe.Pointer(&h))
}
