package law

import "sync"

type Element struct {
	buffer []byte

	updateAt int64
}

func NewElement() *Element {

	return &Element{}

}

func (e *Element) Reset() {

	e.buffer = nil

	e.updateAt = 0

}

type ElementPool struct {
	pool *sync.Pool
}

func NewElementPool() *ElementPool {

	return &ElementPool{

		pool: &sync.Pool{

			New: func() interface{} {

				return NewElement()

			},
		},
	}
}

func (ep *ElementPool) Get() *Element {

	return ep.pool.Get().(*Element)

}

func (ep *ElementPool) Put(e *Element) {

	if e != nil {

		e.Reset()

		ep.pool.Put(e)
	}
}
