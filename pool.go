package rio

// The pool is a list of workers. The pool is also a priority queue.
type Pool []*Worker

func (p Pool) Len() int {
	return len(p)
}

func (p Pool) Less(i, j int) bool {
	return p[i].pending < p[j].pending
}

func (p *Pool) Swap(i, j int) {
	(*p)[i], (*p)[j] = (*p)[j], (*p)[i]
}

func (p *Pool) Push(x interface{}) {
	//n := len(*p)
	item := x.(*Worker)
	//item.index = n
	*p = append(*p, item)
}

func (p *Pool) Pop() interface{} {
	old := *p
	n := len(old)
	item := old[n-1]
	//item.index = 0 // for safety
	*p = old[0 : n-1]
	return item
}
