package pool

// NewFunc is a function type that creates a new instance of an object.
type NewFunc = func() interface{}

// StackInterface is an interface that defines the behavior of a stack.
type StackInterface interface {
	Push(value interface{})
	Pop() interface{}
	Len() uint64
}

// Pool is a struct that represents an object pool.
type Pool struct {
	newFunc NewFunc
	stack   StackInterface
}

// NewPool creates a new object pool with the given newFunc and stack.
func NewPool(newFunc NewFunc, stack StackInterface) *Pool {
	p := Pool{
		newFunc: newFunc,
		stack:   stack,
	}

	return &p
}

// Get retrieves an object from the pool. If the pool is empty, a new object is created using newFunc.
func (p *Pool) Get() interface{} {
	v := p.stack.Pop()
	if v == nil {
		return p.newFunc()
	}
	return v
}

// Put puts an object back into the pool.
func (p *Pool) Put(v interface{}) {
	p.stack.Push(v)
}

// Prune removes a portion of the objects from the pool.
func (p *Pool) Prune() {
	count := uint64(float64(p.stack.Len()) * 0.33)
	for i := uint64(0); i < count; i++ {
		_ = p.stack.Pop()
	}
}
