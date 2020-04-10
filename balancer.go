package rio

import (
	"container/heap"
	"fmt"
	"log"
	"time"
)

type Balancer struct {
	pool         Pool
	jobChannel   chan *Request
	done         chan *Worker
	queuedItems  int
	closeChannel chan chan bool
}

func GetBalancer(workerCount, taskPerWorker int) *Balancer {
	b := &Balancer{
		done:         make(chan *Worker),
		jobChannel:   make(chan *Request),
		closeChannel: make(chan chan bool),
	}
	p := make([]*Worker, workerCount)
	for i := 0; i < workerCount; i++ {
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
	b.balance()
	return b
}

func (b *Balancer) PostJob(job *Request) {
	b.jobChannel <- job
}

func (b *Balancer) Close(cb chan bool) {
	b.closeChannel <- cb
}

func (b *Balancer) balance() {
	go func() {
		for {
			select {
			case req := <-b.jobChannel:
				b.dispatch(req)
				b.queuedItems++
			case w := <-b.done:
				b.completed(w)
				b.queuedItems--
			case cb := <-b.closeChannel:
				if b.queuedItems > 0 {
					time.AfterFunc(1*time.Second, func() { b.closeChannel <- cb })
				} else {
					cb <- true
					log.Println("Closing balancer")
					return
				}
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
