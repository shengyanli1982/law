package pool

type NewFunc = func() any

type QueueInterface interface {
	Push(value any)
	Pop() any
}

type Pool struct {
	newFunc NewFunc
	queue   QueueInterface
}

func NewPool(newFunc NewFunc, queue QueueInterface) *Pool {
	p := Pool{
		newFunc: newFunc,
		queue:   queue,
	}

	return &p
}

func (p *Pool) Get() any {
	v := p.queue.Pop()
	if v == nil {
		return p.newFunc()
	}
	return v
}

func (p *Pool) Put(v any) {
	p.queue.Push(v)
}
