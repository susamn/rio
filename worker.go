package rio

import (
	"log"
	"time"
)

type Worker struct {
	Name         string
	requests     chan *Request
	pending      int
	index        int
	done         chan *Worker
	closeChannel chan chan bool
}

func (w *Worker) DoWork(request *Request) {
	w.requests <- request
}

func (w *Worker) Close(cb chan bool) {
	w.closeChannel <- cb
}

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

				// No task to work on
				if len(r.Tasks) == 0 {
					return
				}
				if len(r.Tasks) > 1 && len(r.Bridges) != len(r.Tasks)-1 {
					log.Println("If you are specifying multiple tasks, n, then the you must provide (n-1) bridges")
					log.Printf("Provided task count : %d, bridge count : %d. Expected bridge count : %d\n", len(r.Tasks), len(r.Bridges), len(r.Tasks)-1)
					return
				}

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

func doTask(ch chan *Response, task *FutureTask, bridgeConnection *BridgeConnection) {
	go func() {
		preTime := time.Now()
		// The actual network call happens here
		futureTaskResponse := task.Callback(bridgeConnection)
		ch <- &Response{
			ResponseTime: time.Since(preTime),
			ResponseCode: futureTaskResponse.ResponseCode,
			Data:         futureTaskResponse.Data,
			Error:        futureTaskResponse.Error,
		}
	}()
}
