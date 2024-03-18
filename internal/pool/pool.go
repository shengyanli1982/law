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
	// newFunc 是一个函数，用于创建新的对象实例。
	// newFunc is a function used to create new object instances.
	newFunc NewFunc

	// stack 是一个栈，用于存储对象实例。
	// stack is a stack used to store object instances.
	stack StackInterface
}

// NewPool 创建一个具有给定newFunc和stack的新对象池。
// NewPool creates a new object pool with the given newFunc and stack.
func NewPool(newFunc NewFunc, stack StackInterface) *Pool {
	p := Pool{
		// 设置创建新对象实例的函数。
		// Set the function to create new object instances.
		newFunc: newFunc,

		// 设置存储对象实例的栈。
		// Set the stack to store object instances.
		stack: stack,
	}

	// 返回新创建的对象池。
	// Return the newly created object pool.
	return &p
}

// 定义一个名为 Get 的方法，该方法用于从 Pool 中获取一个对象
// Define a method named Get, which is used to get an object from the Pool
func (p *Pool) Get() interface{} {
	// 从 Pool 的 stack 中弹出一个对象
	// Pop an object from the stack of the Pool
	v := p.stack.Pop()

	// 如果从 stack 中弹出的对象为空，则使用 Pool 的 newFunc 创建一个新的对象
	// If the object popped from the stack is null, create a new object using newFunc of the Pool
	if v == nil {
		return p.newFunc()
	}

	// 返回从 Pool 中取出的对象
	// Return the object taken from the Pool
	return v
}

// Put 将一个对象放回池中。
// Put puts an object back into the pool.
func (p *Pool) Put(v interface{}) {
	// 将对象推入栈中，即放回池中。
	// Push the object into the stack, i.e., put it back into the pool.
	p.stack.Push(v)
}

// Prune 是一个方法，用于从 Pool 中删除一部分对象，默认是总数的 33%
// Prune is a method used to remove a portion of the objects from the Pool, default is 33% of the total
func (p *Pool) Prune() {
	// 计算要删除的对象数量，即总数的33%
	// Calculate the number of objects to be deleted, i.e., 33% of the total
	count := uint64(float64(p.stack.Len()) * 0.33)

	// 使用 for 循环来删除指定数量的对象
	// Use a for loop to delete the specified number of objects
	for i := uint64(0); i < count; i++ {
		// 从 Pool 的 stack 中弹出并丢弃对象，即删除对象
		// Pop and discard objects from the stack of the Pool, i.e., delete objects
		_ = p.stack.Pop()
	}
}
