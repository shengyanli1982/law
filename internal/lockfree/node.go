package lockfree

import (
	"sync/atomic"
	"unsafe"
)

// Node 数据单元节点
// Node represents a data unit node
type Node struct {
	// value 是节点存储的值，类型为 interface{}，可以存储任何类型的值
	// value is the value stored in the node, of type interface{}, which can store any type of value
	value interface{}

	// next 是指向下一个节点的指针，类型为 unsafe.Pointer
	// next is a pointer to the next node, of type unsafe.Pointer
	next unsafe.Pointer
}

// NewNode 函数用于创建一个新的 Node 结构体实例
// The NewNode function is used to create a new instance of the Node struct
func NewNode() *Node {
	// 返回一个新的 Node 结构体实例
	// Returns a new instance of the Node struct
	return &Node{}
}

// Reset 方法用于重置 Node 结构体的值
// The Reset method is used to reset the value of the Node struct
func (n *Node) Reset() {
	// 将 value 字段设置为 nil
	// Set the value field to nil
	n.value = nil

	// 将 next 字段设置为 nil
	// Set the next field to nil
	n.next = nil
}

// loadNode 函数用于加载指定指针 p 指向的 Node 结构体
// The loadNode function is used to load the Node struct pointed to by the specified pointer p
func loadNode(p *unsafe.Pointer) *Node {
	// 使用 atomic.LoadPointer 加载并返回指定指针 p 指向的 Node 结构体
	// Uses atomic.LoadPointer to load and return the Node struct pointed to by the specified pointer p
	return (*Node)(atomic.LoadPointer(p))
}

// compareAndSwapNode 函数用于比较并交换指定指针 p 指向的 Node 结构体
// The compareAndSwapNode function is used to compare and swap the Node struct pointed to by the specified pointer p
func compareAndSwapNode(p *unsafe.Pointer, old, new *Node) bool {
	// 使用 atomic.CompareAndSwapPointer 比较并交换指定指针 p 指向的 Node 结构体
	// Uses atomic.CompareAndSwapPointer to compare and swap the Node struct pointed to by the specified pointer p
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}
