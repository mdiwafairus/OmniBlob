package worker

type Pool struct{}

func NewPool(size int) *Pool {
	return &Pool{}
}

func (p *Pool) Start() {}
