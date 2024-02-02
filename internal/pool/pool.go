package pool

// NewFunc 是 一个函数类型，用于创建一个新的对象实例
// NewFunc is a function type that creates a new instance of an object.
type NewFunc = func() interface{}

// StackInterface 是一个定义了栈行为的接口
// StackInterface is an interface that defines the behavior of a stack.
type StackInterface interface {
	Push(value interface{})
	Pop() interface{}
	Len() uint64
}

// Pool 是一个表示对象池的结构
// Pool is a struct that represents an object pool.
type Pool struct {
	newFunc NewFunc
	stack   StackInterface
}

// NewPool 创建一个具有给定newFunc和stack的新对象池
// NewPool creates a new object pool with the given newFunc and stack.
func NewPool(newFunc NewFunc, stack StackInterface) *Pool {
	p := Pool{
		newFunc: newFunc,
		stack:   stack,
	}

	return &p
}

// Get 从池中获得一个对象。如果池为空，则使用newFunc创建一个新对象。
// Get retrieves an object from the pool. If the pool is empty, a new object is created using newFunc.
func (p *Pool) Get() interface{} {
	v := p.stack.Pop()
	if v == nil {
		return p.newFunc()
	}
	return v
}

// Put 将一个对象放回池中。
// Put puts an object back into the pool.
func (p *Pool) Put(v interface{}) {
	p.stack.Push(v)
}

// Prune 从池中删除一部分对象。 默认是总数的 33%
// Prune removes a portion of the objects from the pool. Default is 33% of the total.
func (p *Pool) Prune() {
	count := uint64(float64(p.stack.Len()) * 0.33)
	for i := uint64(0); i < count; i++ {
		_ = p.stack.Pop()
	}
}
