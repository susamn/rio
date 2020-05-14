package rio

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

// This is a response which will be available in future from a FutureTask
type FutureTaskResponse struct {
	ResponseCode int
	Data         interface{}
	Error        error
}

// During callback chaining, ue this to setup the callbacks, see the example
var EMPTY_ARG_PLACEHOLDER = ""

// Use this to send an empty response when some callback is failed
var EMPTY_CALLBACK_RESPONSE = &FutureTaskResponse{
	ResponseCode: -1,
	Data:         nil,
	Error:        errors.New("The callback didn't run due to argument unavailability"),
}

// This is task which will be executed in future
type FutureTask struct {
	Name         string
	Callback     Callback
	Timeout      time.Duration
	RetryCount   int
	ReplicaCount int
}

// Its how two callbacks communicate with each other, this is a function which knows how to convert
// one callback response to the next
type Bridge func(interface{}) *BridgeConnection

// Its the type that will be used by the consumers to create the service closures
type Callback func(*BridgeConnection) *FutureTaskResponse

// Its the data that is filled with the bridge data
type BridgeConnection struct {
	Data  []interface{}
	Error error
}

// Request is the one that is sent to the *balancer* to be used to call concurrently
type Request struct {
	Tasks            []*FutureTask
	Bridges          []Bridge
	Responses        []*Response
	CompletedChannel chan bool
	Ctx              context.Context
}

// Response is the one that is sent to the graphql layer to be sent to the caller
type Response struct {
	ResponseTime time.Duration
	ResponseCode int
	Data         interface{}
	Error        error
}

// GetResponse method gives the response from the request, based on index, use this method, when there are multiple
// tasks sent to the balancer to be processed. When the execution is done, the corresponding response for the queued
// task is available, the same succession.
func (r *Request) GetResponse(index int) (*Response, error) {
	if r.Responses != nil && len(r.Responses) > 0 {
		if index > len(r.Responses)-1 {
			return nil, errors.New(fmt.Sprintf("No response available at index position : %d", index))
		} else {
			return r.Responses[index], nil
		}
	} else {
		return nil, errors.New("No response obtained from the process, the response slice is empty.")
	}
}

// GetOnlyResponse is used to get the one and only response from the request object, use this when there is only 1 task
func (r *Request) GetOnlyResponse() (*Response, error) {
	if len(r.Responses) > 0 {
		return r.Responses[0], nil
	} else {
		return nil, errors.New("No response obtained from the process, the response slice is empty.")
	}
}

// Use this method to create a new task. It takes a callback in the form of a closure.
func NewFutureTask(callback Callback) *FutureTask {
	return &FutureTask{Callback: callback}
}

// Use this method to create a new task. It takes a callback in the form of a closure.
func NewNamedFutureTask(name string, callback Callback) *FutureTask {
	return &FutureTask{Callback: callback, Name: name}
}

// Add timeout for the task in the form of milliseconds
func (f *FutureTask) WithMilliSecondTimeout(t int) *FutureTask {
	f.Timeout = time.Duration(t) * time.Millisecond
	return f
}

// Add timeout for the task in the form of seconds
func (f *FutureTask) WithSecondTimeout(t int) *FutureTask {
	f.Timeout = time.Duration(t) * time.Second
	return f
}

// Add retry count to the task. If the task fails, it will be retried this many times. The failure information, comes
// from the task itself.
func (f *FutureTask) WithRetry(c int) *FutureTask {
	f.RetryCount = c
	return f
}

// Add replica calls. Use this when there is a possibility to get different response time from a service for successive
// calls and only the fastest one is needed. The worker will use call the service concurrently, this many times and only
// the fastest will be picked.
func (f *FutureTask) WithReplica(c int) *FutureTask {
	f.ReplicaCount = c
	return f
}

// Use this method to build a request instance, which is sent on the balancer to be processed. Use this variant when
// there is a job chaining required and multiple tasks are involved, one after another.
func BuildRequests(context context.Context, task *FutureTask) *Request {
	tasks := make([]*FutureTask, 0, 1)
	tasks = append(tasks, task)
	return &Request{Ctx: context, Tasks: tasks, CompletedChannel: make(chan bool)}
}

// This method validates the posted job/request to the balancer. If validation fails, balancer sends the error to the
// calling goroutine immediately, otherwise sends the request to the workers.
func (r Request) Validate() error {
	if r.CompletedChannel == nil {
		return errors.New("The request CompletedChannel is nil")
	}
	if r.Tasks == nil || len(r.Tasks) == 0 {
		return errors.New("please provide some tasks to process, the task list is empty")
	}
	if length := len(r.Tasks); length > 1 && length != len(r.Bridges)+1 {
		return errors.New("for a followed by construct, there should be n requests and (n-1) bridges")
	}
	if len(r.Tasks) > 1 && len(r.Bridges) != len(r.Tasks)-1 {
		log.Println("If you are specifying multiple tasks, n, then the you must provide (n-1) bridges")
		return errors.New(fmt.Sprintf("Provided task count : %d, bridge count : %d. Expected "+
			"bridge count : %d\n", len(r.Tasks), len(r.Bridges), len(r.Tasks)-1))
	}
	return nil
}

// This construct is used to create task chaining. If task2 depends on task1 in terms of data and the execution is to
// happen like task1-->task2, then use this method to chain them together by means of a Bridge type
func (r *Request) FollowedBy(bridge Bridge, task *FutureTask) *Request {
	if bridge == nil || task == nil {
		log.Println("Error : Please provide the bridges and tasks properly")
		return nil
	}
	if r.Bridges == nil {
		r.Bridges = make([]Bridge, 0, 1)
	}
	r.Bridges = append(r.Bridges, bridge)
	r.Tasks = append(r.Tasks, task)
	return r
}
