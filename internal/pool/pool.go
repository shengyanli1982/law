package pool

// NewFunc 是一个函数类型，用于创建一个新的对象实例。
// NewFunc is a function type that creates a new instance of an object.
type NewFunc = func() interface{}

// StackInterface 是一个定义了栈行为的接口。
// StackInterface is an interface that defines the behavior of a stack.
type StackInterface interface {
	// Push 方法接受一个值，尝试将其推入栈。
	// The Push method takes a value and attempts to push it into the stack.
	Push(value interface{})

	// Pop 方法尝试从栈中弹出一个值。
	// The Pop method attempts to pop a value from the stack.
	Pop() interface{}

	// Len 方法返回栈的长度。
	// The Len method returns the length of the stack.
	Len() uint64
}

// Pool 是一个表示对象池的结构。
// Pool is a struct that represents an object pool.
type Pool struct {
	newFunc NewFunc        // newFunc 是一个函数，用于创建新的对象实例。
	stack   StackInterface // stack 是一个栈，用于存储对象实例。
}

// NewPool 创建一个具有给定newFunc和stack的新对象池。
// NewPool creates a new object pool with the given newFunc and stack.
func NewPool(newFunc NewFunc, stack StackInterface) *Pool {
	p := Pool{
		newFunc: newFunc, // 设置创建新对象实例的函数。
		stack:   stack,   // 设置存储对象实例的栈。
	}

	return &p // 返回新创建的对象池。
}

// Get 从池中获得一个对象。如果池为空，则使用newFunc创建一个新对象。
// Get retrieves an object from the pool. If the pool is empty, a new object is created using newFunc.
func (p *Pool) Get() interface{} {
	v := p.stack.Pop() // 尝试从栈中弹出一个对象。
	if v == nil {      // 如果弹出的对象为空，即栈为空。
		return p.newFunc() // 使用 newFunc 创建一个新的对象。
	}
	return v // 返回从栈中弹出的对象。
}

// Put 将一个对象放回池中。
// Put puts an object back into the pool.
func (p *Pool) Put(v interface{}) {
	p.stack.Push(v) // 将对象推入栈中。
}

// Prune 从池中删除一部分对象。 默认是总数的 33%。
// Prune removes a portion of the objects from the pool. Default is 33% of the total.
func (p *Pool) Prune() {
	count := uint64(float64(p.stack.Len()) * 0.33) // 计算需要删除的对象数量，即栈的长度的 33%。
	for i := uint64(0); i < count; i++ {           // 循环删除对象。
		_ = p.stack.Pop() // 从栈中弹出一个对象。
	}
}
