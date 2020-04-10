package rio

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func BenchmarkBalancerSingleTask(b *testing.B) {

	balancer := GetBalancer(1, 1)
	for i := 0; i < b.N; i++ {

		var tasks = make([]*FutureTask, 1)
		tasks[0] = &FutureTask{Callback: Task3, Timeout: time.Duration(1) * time.Second, RetryCount: 0}

		completeChannel := make(chan bool)

		ctx := context.Background()

		request := &Request{
			Tasks:            tasks,
			Bridges:          nil,
			Responses:        nil,
			CompletedChannel: completeChannel,
			Ctx:              ctx,
		}

		balancer.PostJob(request)

		<-request.CompletedChannel

		fmt.Println(request.Responses[0])

	}
}

func BenchmarkMultipleChainedTask(b *testing.B) {
	balancer := GetBalancer(10, 2)
	for i := 0; i < b.N; i++ {

		var tasks = make([]*FutureTask, 4)
		tasks[0] = &FutureTask{Callback: Task1, Timeout: time.Duration(100) * time.Second, RetryCount: 2}
		tasks[1] = &FutureTask{Callback: Task2, Timeout: time.Duration(100) * time.Second, RetryCount: 0}
		tasks[2] = &FutureTask{Callback: Task3, Timeout: time.Duration(100) * time.Second, RetryCount: 1}
		tasks[3] = &FutureTask{Callback: Task4, Timeout: time.Duration(100) * time.Second}

		var bridges = make([]Bridge, 3)
		bridges[0] = Bridge1
		bridges[1] = Bridge2
		bridges[2] = Bridge3

		completeChannel := make(chan bool)

		ctx := context.Background()

		request := &Request{
			Tasks:            tasks,
			Bridges:          bridges,
			Responses:        nil,
			CompletedChannel: completeChannel,
			Ctx:              ctx,
		}

		balancer.PostJob(request)

		<-request.CompletedChannel

		fmt.Println(request.Responses[0], request.Responses[1], request.Responses[2], request.Responses[3])

		if request.Responses[0].Data.(string) != "Response 1" ||
			request.Responses[1].Data.(string) != "Response 2" ||
			request.Responses[2].Data.(string) != "Response 3" ||
			request.Responses[3].Data.(string) != "Response 4" {

		}
	}
}

func TestWithSingleTaskWithRetry(t *testing.T) {
	balancer := GetBalancer(1, 1)

	var tasks = make([]*FutureTask, 1)
	tasks[0] = &FutureTask{Callback: Task1, Timeout: time.Duration(100) * time.Second, RetryCount: 2}

	var bridges = make([]Bridge, 1)
	bridges[0] = Bridge4

	completeChannel := make(chan bool)

	ctx := context.Background()

	request := &Request{
		Tasks:            tasks,
		Bridges:          bridges,
		Responses:        nil,
		CompletedChannel: completeChannel,
		Ctx:              ctx,
	}

	balancer.PostJob(request)

	<-request.CompletedChannel

	fmt.Println(request.Responses[0])

	closeChannel := make(chan bool)
	balancer.Close(closeChannel)
	<-closeChannel

}

func TestWithSingleTaskWithTimeout(t *testing.T) {

	balancer := GetBalancer(1, 1)

	var tasks = make([]*FutureTask, 1)
	tasks[0] = &FutureTask{Callback: Task7, Timeout: time.Duration(2) * time.Second, RetryCount: 2}

	completeChannel := make(chan bool)

	ctx := context.Background()

	request := &Request{
		Tasks:            tasks,
		Bridges:          nil,
		Responses:        nil,
		CompletedChannel: completeChannel,
		Ctx:              ctx,
	}

	balancer.PostJob(request)

	<-request.CompletedChannel

	_, err := request.GetResponse(0)

	if err == nil {
		t.Fail()
	}

	closeChannel := make(chan bool)
	balancer.Close(closeChannel)
	<-closeChannel

}

func TestWithSingleTaskWithContextCancel(t *testing.T) {
	balancer := GetBalancer(1, 1)

	var tasks = make([]*FutureTask, 1)
	tasks[0] = &FutureTask{Callback: Task7, Timeout: time.Duration(20) * time.Second, RetryCount: 2}

	completeChannel := make(chan bool)

	ctx, cancel := context.WithCancel(context.Background())

	request := &Request{
		Tasks:            tasks,
		Bridges:          nil,
		Responses:        nil,
		CompletedChannel: completeChannel,
		Ctx:              ctx,
	}

	balancer.PostJob(request)

	go func() {
		time.Sleep(time.Duration(4) * time.Second)
		cancel()
	}()

	<-request.CompletedChannel

	_, err := request.GetResponse(0)

	if err == nil {
		t.Fail()
	}

	closeChannel := make(chan bool)
	balancer.Close(closeChannel)
	<-closeChannel

}

func TestWithMultipleChainedTasks(t *testing.T) {

	balancer := GetBalancer(10, 2)

	var tasks = make([]*FutureTask, 4)
	tasks[0] = &FutureTask{Callback: Task1, Timeout: time.Duration(100) * time.Second, RetryCount: 2}
	tasks[1] = &FutureTask{Callback: Task2, Timeout: time.Duration(100) * time.Second, RetryCount: 0}
	tasks[2] = &FutureTask{Callback: Task3, Timeout: time.Duration(100) * time.Second, RetryCount: 1}
	tasks[3] = &FutureTask{Callback: Task4, Timeout: time.Duration(100) * time.Second}

	var bridges = make([]Bridge, 3)
	bridges[0] = Bridge1
	bridges[1] = Bridge2
	bridges[2] = Bridge3

	completeChannel := make(chan bool)

	ctx := context.Background()

	request := &Request{
		Tasks:            tasks,
		Bridges:          bridges,
		Responses:        nil,
		CompletedChannel: completeChannel,
		Ctx:              ctx,
	}

	balancer.PostJob(request)

	<-request.CompletedChannel

	fmt.Println(request.Responses[0], request.Responses[1], request.Responses[2], request.Responses[3])

	if request.Responses[0].Data.(string) != "Response 1" ||
		request.Responses[1].Data.(string) != "Response 2" ||
		request.Responses[2].Data.(string) != "Response 3" ||
		request.Responses[3].Data.(string) != "Response 4" {
		t.Fail()

	}

	closeChannel := make(chan bool)
	balancer.Close(closeChannel)
	<-closeChannel

}

func TestWithMultipleChainedTasksWithThirdTaskTimedOut(t *testing.T) {

	balancer := GetBalancer(10, 2)

	var tasks = make([]*FutureTask, 4)
	tasks[0] = &FutureTask{Callback: Task1, Timeout: time.Duration(100) * time.Second, RetryCount: 2}
	tasks[1] = &FutureTask{Callback: Task2, Timeout: time.Duration(100) * time.Second, RetryCount: 0}
	tasks[2] = &FutureTask{Callback: Task7, Timeout: time.Duration(3) * time.Second, RetryCount: 1}
	tasks[3] = &FutureTask{Callback: Task4, Timeout: time.Duration(100) * time.Second}

	var bridges = make([]Bridge, 3)
	bridges[0] = Bridge1
	bridges[1] = Bridge2
	bridges[2] = Bridge3

	completeChannel := make(chan bool)

	ctx := context.Background()

	request := &Request{
		Tasks:            tasks,
		Bridges:          bridges,
		Responses:        nil,
		CompletedChannel: completeChannel,
		Ctx:              ctx,
	}

	balancer.PostJob(request)

	<-request.CompletedChannel

	fmt.Println(request.Responses[0], request.Responses[1])

	_, err := request.GetResponse(2)

	if err == nil {
		t.Fail()
	}

	closeChannel := make(chan bool)
	balancer.Close(closeChannel)
	<-closeChannel

}

func TestWithMultipleChainedTaskAndBridgeData(t *testing.T) {
	balancer := GetBalancer(1, 1)

	var tasks = make([]*FutureTask, 3)
	tasks[0] = &FutureTask{Callback: Task1, Timeout: time.Duration(100) * time.Second, RetryCount: 2}
	tasks[1] = &FutureTask{Callback: Task5, Timeout: time.Duration(100) * time.Second, RetryCount: 0}
	tasks[2] = &FutureTask{Callback: Task6, Timeout: time.Duration(100) * time.Second, RetryCount: 0}

	var bridges = make([]Bridge, 2)
	bridges[0] = Bridge4
	bridges[1] = Bridge5

	completeChannel := make(chan bool)

	ctx := context.Background()

	request := &Request{
		Tasks:            tasks,
		Bridges:          bridges,
		Responses:        nil,
		CompletedChannel: completeChannel,
		Ctx:              ctx,
	}

	balancer.PostJob(request)

	<-request.CompletedChannel

	r1, _ := request.GetResponse(0)
	r2, _ := request.GetResponse(1)
	r3, _ := request.GetResponse(2)

	if r1.Data.(string) != "Response 1" ||
		len(r2.Data.([]interface{})) != 3 ||
		len(r3.Data.([]interface{})) != 2 {
		t.Fail()
	}

	closeChannel := make(chan bool)
	balancer.Close(closeChannel)
	<-closeChannel

}

func TestWithMultipleChainedTaskAndBridgeDataFromDifferentGoroutines(t *testing.T) {
	balancer := GetBalancer(100, 100)

	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				var tasks = make([]*FutureTask, 3)
				tasks[0] = &FutureTask{Callback: Task1, Timeout: time.Duration(100) * time.Second, RetryCount: 2}
				tasks[1] = &FutureTask{Callback: Task5, Timeout: time.Duration(100) * time.Second, RetryCount: 0}
				tasks[2] = &FutureTask{Callback: Task6, Timeout: time.Duration(100) * time.Second, RetryCount: 0}

				var bridges = make([]Bridge, 2)
				bridges[0] = Bridge4
				bridges[1] = Bridge5

				completeChannel := make(chan bool)

				ctx := context.Background()

				request := &Request{
					Tasks:            tasks,
					Bridges:          bridges,
					Responses:        nil,
					CompletedChannel: completeChannel,
					Ctx:              ctx,
				}

				balancer.PostJob(request)

				<-request.CompletedChannel

				r1, _ := request.GetResponse(0)
				r2, _ := request.GetResponse(1)
				r3, _ := request.GetResponse(2)

				if r1.Data.(string) != "Response 1" ||
					len(r2.Data.([]interface{})) != 3 ||
					len(r3.Data.([]interface{})) != 2 {
					t.Fail()
				}
			}
		}()
	}

	closeChannel := make(chan bool)
	balancer.Close(closeChannel)
	<-closeChannel

}

func TestWithMultipleChainedTaskAndBridgeDataSecondCallFailed(t *testing.T) {
	balancer := GetBalancer(1, 1)

	var tasks = make([]*FutureTask, 4)
	tasks[0] = &FutureTask{Callback: Task1, Timeout: time.Duration(100) * time.Second, RetryCount: 2}
	tasks[1] = &FutureTask{Callback: Task5, Timeout: time.Duration(100) * time.Second, RetryCount: 0}
	tasks[2] = &FutureTask{Callback: Task6, Timeout: time.Duration(100) * time.Second, RetryCount: 0}
	tasks[3] = &FutureTask{Callback: Task7, Timeout: time.Duration(100) * time.Second, RetryCount: 0}

	var bridges = make([]Bridge, 3)
	bridges[0] = Bridge6
	bridges[1] = Bridge7
	bridges[2] = Bridge8

	completeChannel := make(chan bool)

	context := context.Background()

	request := &Request{
		Tasks:            tasks,
		Bridges:          bridges,
		Responses:        nil,
		CompletedChannel: completeChannel,
		Ctx:              context,
	}

	balancer.PostJob(request)

	<-request.CompletedChannel

	r1, _ := request.GetResponse(0)
	r2, _ := request.GetResponse(1)
	r3, _ := request.GetResponse(2)
	r4, _ := request.GetResponse(3)

	fmt.Println(r1, r2, r3, r4)

	if r3.Error.Error() != "Test Error" || r4.Error.Error() != "Test Error" {
		t.Fail()
	}

	closeChannel := make(chan bool)
	balancer.Close(closeChannel)
	<-closeChannel

}

func Bridge1(interface{}) *BridgeConnection {
	return &BridgeConnection{}
}

func Bridge2(interface{}) *BridgeConnection {
	return &BridgeConnection{}
}

func Bridge3(interface{}) *BridgeConnection {
	return &BridgeConnection{}
}

func Bridge4(interface{}) *BridgeConnection {
	return &BridgeConnection{
		Data:  []interface{}{"1", 2, 3.0},
		Error: nil,
	}
}

func Bridge5(interface{}) *BridgeConnection {
	return &BridgeConnection{
		Data:  []interface{}{"1", "2"},
		Error: nil,
	}
}

func Bridge6(interface{}) *BridgeConnection {
	return &BridgeConnection{
		Data:  []interface{}{"1", 2, 3.0},
		Error: nil,
	}
}

func Bridge7(interface{}) *BridgeConnection {
	return &BridgeConnection{
		Data:  []interface{}{"1", "2"},
		Error: errors.New("Test Error"),
	}
}

func Bridge8(interface{}) *BridgeConnection {
	return &BridgeConnection{
		Data:  []interface{}{"1", "2"},
		Error: nil,
	}
}

func Task1(*BridgeConnection) *FutureTaskResponse {
	fmt.Print("Task 1-->")
	return &FutureTaskResponse{
		ResponseCode: 404,
		Data:         "Response 1",
		Error:        errors.New(""),
	}
}

func Task2(*BridgeConnection) *FutureTaskResponse {

	fmt.Println("Task 2-->")
	return &FutureTaskResponse{
		ResponseCode: 500,
		Data:         "Response 2",
		Error:        nil,
	}
}

func Task3(*BridgeConnection) *FutureTaskResponse {
	fmt.Print("Task 3-->")
	return &FutureTaskResponse{
		ResponseCode: 200,
		Data:         "Response 3",
		Error:        errors.New(""),
	}
}

func Task4(*BridgeConnection) *FutureTaskResponse {
	fmt.Println("Task 4-->")
	return &FutureTaskResponse{
		ResponseCode: 404,
		Data:         "Response 4",
		Error:        nil,
	}
}

func Task5(bconn *BridgeConnection) *FutureTaskResponse {
	fmt.Print("Task 5-->")
	d1 := bconn.Data[0].(string)
	d2 := bconn.Data[1].(int)
	d3 := bconn.Data[2].(float64)
	return &FutureTaskResponse{
		ResponseCode: 200,
		Data:         []interface{}{d1, d2, d3},
		Error:        nil,
	}
}

func Task6(bconn *BridgeConnection) *FutureTaskResponse {
	fmt.Println("Task 6-->")
	d1 := bconn.Data[0].(string)
	d2 := bconn.Data[1].(string)
	return &FutureTaskResponse{
		ResponseCode: 200,
		Data:         []interface{}{d1, d2},
		Error:        nil,
	}
}

func Task7(*BridgeConnection) *FutureTaskResponse {
	fmt.Print("Task 7-->")
	time.Sleep(time.Duration(5) * time.Second)
	return &FutureTaskResponse{
		ResponseCode: 404,
		Data:         "Response 7",
		Error:        errors.New(""),
	}
}
