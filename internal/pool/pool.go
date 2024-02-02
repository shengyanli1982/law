package pool

type NewFunc = func() any

type StackInterface interface {
	Push(value any)
	Pop() any
	Len() uint64
}

type Pool struct {
	newFunc NewFunc
	stack   StackInterface
}

func NewPool(newFunc NewFunc, queue StackInterface) *Pool {
	p := Pool{
		newFunc: newFunc,
		stack:   queue,
	}

	return &p
}

func (p *Pool) Get() any {
	v := p.stack.Pop()
	if v == nil {
		return p.newFunc()
	}
	return v
}

func (p *Pool) Put(v any) {
	p.stack.Push(v)
}

func (p *Pool) Prune() {
	count := uint64(float64(p.stack.Len()) * 0.33)
	for i := uint64(0); i < count; i++ {
		_ = p.stack.Pop()
	}
}
