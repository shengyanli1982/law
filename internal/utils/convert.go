package utils

import "unsafe"

// BytesToString 是一个函数，它将字节切片转换为字符串，而不进行任何内存分配。
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytes 是一个函数，它将字符串转换为字节切片，而不进行任何内存分配。
func StringToBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}
