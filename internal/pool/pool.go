package pool

type NewFunc = func() interface{}

type StackInterface interface {
	Push(value interface{})

	Pop() interface{}

	Len() uint64
}

type Pool struct {
	newFunc NewFunc

	stack StackInterface
}

func NewPool(newFunc NewFunc, stack StackInterface) *Pool {

	p := Pool{

		newFunc: newFunc,

		stack: stack,
	}

	return &p

}

func (p *Pool) Get() interface{} {

	v := p.stack.Pop()

	if v == nil {

		return p.newFunc()

	}

	return v

}

func (p *Pool) Put(v interface{}) {

	p.stack.Push(v)

}

func (p *Pool) Prune() {

	count := uint64(float64(p.stack.Len()) * 0.33)

	for i := uint64(0); i < count; i++ {

		_ = p.stack.Pop()

	}

}
