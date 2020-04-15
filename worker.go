package rio

import (
	"log"
	"time"
)

// The worker struct, it has all the attributes that is needed by a worker to do its thing
type Worker struct {

	// The name of the worker. It is assigned by the balancer when it is created.
	Name string

	// The request channel of the worker. The balancer sends the requests in this channel
	requests chan *Request

	// The is the count that tells how many requests are still in buffer for the worker to work on
	pending int

	// The index value is used by the priority queue to move it back and forth in the heap
	index int

	// Its the copy of the balancer done channel, passed to all the worker
	done chan *Worker

	// Its the close channel to close a worker. Its used by the balancer only, hence unexported
	closeChannel chan chan bool
}

// The balancer calls the method to queue a new request to the worker
func (w *Worker) DoWork(request *Request) {
	w.requests <- request
}

// The close method, when called closes a worker
func (w *Worker) Close(cb chan bool) {
	w.closeChannel <- cb
}

// The run method which actually processes the requests. Once a worker is created, this method is also called by the
// balancer
func (w *Worker) Run() {
	go func() {
		for {
			select {
			case callback := <-w.closeChannel:
				close(w.closeChannel)
				close(w.requests)
				log.Println("Closing worker : ", w.Name)
				callback <- true
				return

			case r := <-w.requests:

				// Create a slice of response with equal size of the number of requests
				r.Responses = make([]*Response, 0, len(r.Tasks))

				// The initial bridge, which is nil for the first call
				var bridgeConnection *BridgeConnection

				// Single request processing channel
				ch := make(chan *Response)

				currentTask := r.Tasks[0]
				currentTimer := time.NewTimer(currentTask.Timeout)
				doTask(ch, currentTask, bridgeConnection)

				w.loop(currentTimer, r, bridgeConnection, currentTask, ch)

			}

		}
	}()
}

// This method handles the individual tasks and its timeout and the request context
func (w *Worker) loop(currentTimer *time.Timer, r *Request, bridgeConnection *BridgeConnection, currentTask *FutureTask, ch chan *Response) {
	for {
		select {
		case <-r.Ctx.Done():
			log.Println("Context cancelled")
			w.done <- w
			r.CompletedChannel <- true
			return
		case <-currentTimer.C:
			log.Println("Timeout")
			w.done <- w
			r.CompletedChannel <- true
			return
		case response := <-ch:
			currentTimer.Stop()
			if len(r.Tasks)-1 == 0 {
				if response.Error != nil && currentTask.RetryCount > 0 {
					currentTask.RetryCount--
					log.Println("Retrying task")
					currentTimer = time.NewTimer(currentTask.Timeout)
					doTask(ch, currentTask, bridgeConnection)
				} else {
					r.Responses = append(r.Responses, response)
					w.done <- w
					r.CompletedChannel <- true
					return
				}

			} else {
				if response.Error != nil && currentTask.RetryCount > 0 {
					currentTask.RetryCount--
					log.Println("Retrying task")
					currentTimer = time.NewTimer(currentTask.Timeout)
					doTask(ch, currentTask, bridgeConnection)
				} else {
					r.Responses = append(r.Responses, response)
					r.Tasks = r.Tasks[1:]
					bridge := r.Bridges[0]
					if len(r.Bridges) > 1 {
						r.Bridges = r.Bridges[1:]
					}
					if bridge == nil {
						log.Printf("Cannot access bridge as it is nil, check your bridge configuration")
						return
					}
					if response.Data == nil {
						log.Printf("Cannot proceed the chain, the response from the parent call is nil")
						return
					}
					bridgeConnection = bridge(response.Data)

					if bridgeConnection.Error == nil {
						currentTask = r.Tasks[0]
						currentTimer = time.NewTimer(currentTask.Timeout)
						doTask(ch, currentTask, bridgeConnection)
					} else {
						for i := 0; i < len(r.Tasks); i++ {
							r.Responses = append(r.Responses, &Response{
								ResponseTime: -1,
								ResponseCode: -1,
								Data:         nil,
								Error:        bridgeConnection.Error,
							})
						}
						w.done <- w
						r.CompletedChannel <- true
						return
					}
				}

			}

		}
	}
}

// This method handles the execution of the actual network call
func doTask(ch chan *Response, task *FutureTask, bridgeConnection *BridgeConnection) {
	// The actual network call happens here
	go func() {
		var futureTaskResponse *FutureTaskResponse
		preTime := time.Now()
		if task.ReplicaCount > 1 {
			replicaChannel := make(chan *FutureTaskResponse)
			for i := 0; i < task.RetryCount; i++ {
				go func() { replicaChannel <- task.Callback(bridgeConnection) }()
			}
			futureTaskResponse = <-replicaChannel
			close(replicaChannel)
		} else {
			futureTaskResponse = task.Callback(bridgeConnection)
		}

		ch <- &Response{
			ResponseTime: time.Since(preTime),
			ResponseCode: futureTaskResponse.ResponseCode,
			Data:         futureTaskResponse.Data,
			Error:        futureTaskResponse.Error,
		}
	}()
}
