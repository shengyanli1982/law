package lockfree

import (
	"sync/atomic"

	"unsafe"
)

type Node struct {
	value interface{}

	next unsafe.Pointer
}

func NewNode() *Node {

	return &Node{}

}

func (n *Node) Reset() {

	n.value = nil

	n.next = nil

}

func loadNode(p *unsafe.Pointer) *Node {

	return (*Node)(atomic.LoadPointer(p))

}

func compareAndSwapNode(p *unsafe.Pointer, old, new *Node) bool {

	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))

}
