package rio

import (
	"container/heap"
	"fmt"
	"log"
	"time"
)

// The balancer struct, this struct is used inside the GetBalancer method to provide a load balancer to the caller
type Balancer struct {

	// Its the pool of Worker, which is itself a priority queue based on min heap.
	pool Pool

	// This channel is used to receive a request instance form the caller. After getting the request it is dispatched
	// to the most lightly loaded worker
	jobChannel chan *Request

	// This channel is used by the worker. After processing a task, a worker uses this channel to let the balancer know
	// that it is done and able to take new requests from its request channel
	done chan *Worker

	// Its the number of queued requests
	queuedItems int

	// The close channel. When the Close method is called by any calling goroutine sending a chanel of boolean, the
	// balancer waits for all the requests to be processed, then closes all the worker, closes all its owen loops and
	// then finally respond by sending boolean true to the passed channel by the caller, confirming that all the inner
	// loop are closed and the balancer is shutdown.
	closeChannel chan chan bool
}

// Use this method to create an instance of the balancer/load balancer. This method must be created only one, per
// the go runtime as it is very much resource intensive.
func GetBalancer(workerCount, taskPerWorker int) *Balancer {
	b := &Balancer{
		done:         make(chan *Worker),
		jobChannel:   make(chan *Request),
		closeChannel: make(chan chan bool),
	}
	p := make([]*Worker, workerCount)
	for i := 0; i < workerCount; i++ {
		w := &Worker{
			requests:     make(chan *Request, taskPerWorker),
			pending:      0,
			index:        i,
			Name:         fmt.Sprintf("Worker-%d", i),
			done:         b.done,
			closeChannel: make(chan chan bool),
		}
		p[i] = w
		w.Run()
	}
	b.pool = p
	b.balance()
	return b
}

// Use this method from the caller side to queue a new job/request. It will be validated and if found proper, will be
// passed to the worker to be processed. This method returns immediately
func (b *Balancer) PostJob(job *Request) error {
	err := job.Validate()
	if err == nil {
		b.jobChannel <- job
		return nil
	}
	return err
}

// Use this method to close/shutdown a balancer. When this is called, balancer waits for all the requests to be
// processed, then closes all the worker, closes all its owen loops and then finally respond by sending boolean true
// to the passed channel by the caller, confirming that all the inner loop are closed and the balancer is shutdown.
// Once shutdown, sending a request to it will raise a panic
func (b *Balancer) Close(cb chan bool) {
	b.closeChannel <- cb
}

// Unexported method. Only used by the balancer to managed the posted requests.
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
					for _, w := range b.pool {
						c := make(chan bool)
						w.Close(c)
						<-c
						fmt.Println("")
					}
					cb <- true
					log.Println("Closing balancer")
					return
				}
			}
		}
	}()

}

// Balancer uses this method to send a validated request to the most lightly loaded worker
func (b *Balancer) dispatch(req *Request) {
	w := heap.Pop(&b.pool).(*Worker)
	log.Println(fmt.Sprintf("Dispatching request to [%s]", w.Name))
	w.DoWork(req)
	w.pending++
	heap.Push(&b.pool, w)
}

// Worker when completes a task return to the balancer and its pending count is decreased by 1
func (b *Balancer) completed(w *Worker) {
	w.pending--
	worker := heap.Remove(&b.pool, w.index)
	heap.Push(&b.pool, worker)
}
