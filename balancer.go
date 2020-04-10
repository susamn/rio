package rio

import (
	"container/heap"
	"fmt"
	"log"
	"sync"
)

var loadbalancer *Balancer
var balancerLock sync.Once

type Balancer struct {
	pool       Pool
	jobChannel chan *Request
	done       chan *Worker
}

func GetBalancer(workerCount, taskPerWorker int) *Balancer {
	balancerLock.Do(func() {
		b := &Balancer{done: make(chan *Worker), jobChannel: make(chan *Request)}
		countOfWorkers := workerCount
		p := make([]*Worker, countOfWorkers)
		for i := 0; i < countOfWorkers; i++ {
			w := &Worker{
				requests: make(chan *Request, taskPerWorker),
				pending:  0,
				index:    i,
				Name:     fmt.Sprintf("Worker-%d", i),
				done:     b.done}
			p[i] = w
			w.Run()
		}
		b.pool = p
		loadbalancer = b
		loadbalancer.balance()
	})
	return loadbalancer
}

func (b *Balancer) PostJob(job *Request) {
	b.jobChannel <- job
}

func (b *Balancer) balance() {
	go func() {
		for {
			select {
			case req := <-b.jobChannel:
				b.dispatch(req)
			case w := <-b.done:
				b.completed(w)
			}
		}
	}()
}

func (b *Balancer) dispatch(req *Request) {
	w := heap.Pop(&b.pool).(*Worker)
	log.Println(fmt.Sprintf("Dispatching request to [%s]", w.Name))
	w.DoWork(req)
	w.pending++
	heap.Push(&b.pool, w)
	log.Println()
}

func (b *Balancer) completed(w *Worker) {
	w.pending--
	heap.Remove(&b.pool, w.index)
	heap.Push(&b.pool, w)
}
