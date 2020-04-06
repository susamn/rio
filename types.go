package rio

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type FutureTaskResponse struct {
	ResponseCode int
	Data         interface{}
	Error        error
}

var EMPTY_CALLBACK_RESPONSE = &FutureTaskResponse{
	ResponseCode: -1,
	Data:         nil,
	Error:        errors.New("The callback didn't run due to argument unavailability"),
}

type FutureTask struct {
	Callback   func(BridgeConnection) *FutureTaskResponse
	Timeout    time.Duration
	RetryCount int
}
type Bridge func(interface{}) BridgeConnection
type BridgeConnection chan interface{}

// Request is the one that is sent to the *balancer* to be used to call concurrently
type Request struct {
	Tasks            []*FutureTask
	TaskCount        int
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

func (r *Request) GetOnlyResponse() (*Response, error) {
	if len(r.Responses) > 0 {
		return r.Responses[0], nil
	} else {
		return nil, errors.New("No response obtained from the process, the response slice is empty.")
	}
}

func NewFutureTask(callback func(BridgeConnection) *FutureTaskResponse) *FutureTask {
	return &FutureTask{Callback: callback}
}
func (f *FutureTask) WithMilliSecondTimeout(t int) *FutureTask {
	f.Timeout = time.Duration(t) * time.Millisecond
	return f
}
func (f *FutureTask) WithSecondTimeout(t int) *FutureTask {
	f.Timeout = time.Duration(t) * time.Second
	return f
}
func (f *FutureTask) WithRetry(c int) *FutureTask {
	f.RetryCount = c
	return f
}

func BuildRequests(context context.Context, task *FutureTask, size int) *Request {
	tasks := make([]*FutureTask, 0, size)
	tasks = append(tasks, task)
	return &Request{Ctx: context, Tasks: tasks, TaskCount: size, CompletedChannel: make(chan bool)}
}

func BuildSingleRequest(context context.Context, task *FutureTask) *Request {
	tasks := make([]*FutureTask, 1)
	tasks = append(tasks, task)
	return &Request{Ctx: context, Tasks: tasks}
}

func (r *Request) FollowedBy(bridge Bridge, task *FutureTask) *Request {
	if r.TaskCount < 2 {
		panic("TaskCount cannot be < 2 for a FollowedBy construct")
	}
	if r.Bridges == nil {
		r.Bridges = make([]Bridge, 0, r.TaskCount-1)
	}

	r.Bridges = append(r.Bridges, bridge)
	r.Tasks = append(r.Tasks, task)

	return r
}
