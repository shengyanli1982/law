package lockfree

import (
	"sync"
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
func NewNode(v interface{}) *Node {
	// 返回一个新的 Node 结构体实例
	// Returns a new instance of the Node struct
	return &Node{value: v}
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

// NodePool 是一个结构体，它包含一个同步池（sync.Pool）的指针。
// NodePool is a struct that contains a pointer to a sync.Pool.
type NodePool struct {
	pool *sync.Pool
}

// NewNodePool 是一个构造函数，它返回一个新的 NodePool 实例。
// NewNodePool is a constructor that returns a new instance of NodePool.
func NewNodePool() *NodePool {
	return &NodePool{
		// 在这里，我们初始化 sync.Pool，并提供一个函数来生成新的 Node 实例。
		// Here, we initialize the sync.Pool and provide a function to generate new Node instances.
		pool: &sync.Pool{
			New: func() interface{} {
				return NewNode(nil)
			},
		},
	}
}

// Get 方法从 NodePool 中获取一个 Node 实例。
// The Get method retrieves a Node instance from the NodePool.
func (p *NodePool) Get() *Node {
	// 我们从 sync.Pool 中获取一个对象，并将其转换为 Node 指针。
	// We get an object from the sync.Pool and cast it to a Node pointer.
	return p.pool.Get().(*Node)
}

// Put 方法将一个 Node 实例放回到 NodePool 中。
// The Put method puts a Node instance back into the NodePool.
func (p *NodePool) Put(n *Node) {
	// 如果 Node 不为 nil，我们将其重置并放回到 sync.Pool 中。
	// If the Node is not nil, we reset it and put it back into the sync.Pool.
	if n != nil {
		n.Reset()
		p.pool.Put(n)
	}
}
