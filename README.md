# Table of Contents

1.  [Introduction](#org8f950e8)
    1.  [What is RIO?](#org64d7944)
    2.  [Concern](#org23a1e51)
        1.  [An asynchronous job processor](#orgf7dc6c9)
        2.  [Easy management of these goroutines and chaining them](#orgc2a657c)


<a id="org8f950e8"></a>

# Introduction


![Go Build](https://github.com/susamn/rio/workflows/Go/badge.svg) [![Coverage](https://codecov.io/gh/susamn/rio/branch/master/graph/badge.svg)](https://codecov.io/gh/susamn/rio) ![license](https://img.shields.io/github/license/susamn/rio)

[![Go Report Card](https://goreportcard.com/badge/github.com/susamn/rio)](https://goreportcard.com/report/github.com/susamn/rio) [![Code Inspector Badge](https://www.code-inspector.com/project/5768/status/svg)](https://www.code-inspector.com/project/5768/status/svg) [![Code Inspector Report Card](https://www.code-inspector.com/project/5768/score/svg)](https://www.code-inspector.com/project/5768/score/svg) [![Codacy Badge](https://api.codacy.com/project/badge/Grade/d833f224d9974475a011cb7191cd19e4)](https://app.codacy.com/manual/susamn/rio?utm_source=github.com&utm_medium=referral&utm_content=susamn/rio&utm_campaign=Badge_Grade_Dashboard)

<a id="org64d7944"></a>

## What is RIO?

Rio is a lightweight job scheduler and job chaining library. Its mainly build for Golang web apps, but it can be very
easily mold to serve any application needing job scheduling. The library is an  asynchronous job processor, which makes
all the backend calls asynchronously with retry, timeout and context cancellation functionality. It also provides very
easy semantics to join multiple datasources based on their output and input types, at the same time having no coupling
between the datasources. This helps in creating new apis or resolvers for GraphQL apis a breeze.


<a id="org23a1e51"></a>

## Concern

Many times we write web apps which connects to different data sources, combines the data obtained from these sources and
then do some more jobs. During these process, we do a lot of boilerplate to transform one data type to other. Also in the
absense of a proper job scheduler, we create goroutines abruptly and without proper management. These create unmanagable
code. To update those code is even more hard in future, when there is a new team member in the team.

Rio tries to solve this problem by introducing two concepts.


<a id="orgf7dc6c9"></a>

### An asynchronous job processor

This is the piece which runs the multiple jobs asynchronously (Based on the Rob Pike video: Google I/O 2010). It has a
priority queue(`balancer.go` and `pool.go`) which hands off incoming requests to a set of managed workers. The balancer
hands off new job to the lightly loaded worker.


<a id="orgc2a657c"></a>

### Easy management of these goroutines and chaining them

How many times do we do this:

    call service 1 in goroutine 1
    wait and get response from goroutine 1
    call service 2 in goroutine 2, taking piece of data from service call 1
    wait and get response from goroutine 2
    call service 3 in goroutine 3, taking piece of data from service call 3
    wait and get response from goroutine 3

You get the idea, this only delays thing more and does a lot of context switching. Rio helps in this, by chaining multiple
calls together by means of using closures and function types and runs in one goroutine.

Now many can think is it not going to be slower compared to doing multiple goroutine calls. I think not, it will be faster.
Think of the previous example. If you do not get response from service 1, can you invoke service 2, or if service 2 fails,
can you call service 3? No right, as there is data dependency between these calls.

Rio chains dependent jobs together by introducing this pattern.

    request := context,
              (<callback of service 1>.WithTimeOut(100 ms).WithRetry(3))
              .FollowedBy(<function which can transform data from service 1 response to request or partial request of 2>,
                          <callback of service 2>)
              .FollowedBy(<function which can transform data from service 2 response to request or partial request of 3>,
                                      <callback of service 3>)

In the example in `examples/web.go` the chaining pattern looks like this:

    request := rio.BuildRequests(context.Background(),
          rio.NewFutureTask(callback1).WithMilliSecondTimeout(10).WithRetry(3), 2).
          FollowedBy(Call1ToCall2, rio.NewFutureTask(callback2).WithMilliSecondTimeout(20))

Once the chaining is done, post the job to load balancer

    balancer.PostJob(request)
    <-request.CompletedChannel

Once the call chain happens, the request comes back with responses for all these calls in a slice and you can do this

1.  Only one job response

        request.GetOnlyResponse()

    or

2.  Multiple job response

        request.GetResponse(index) ---0,1,2

    If any job fails, the response will be empty response, specifically `rio.EMPTY_CALLBACK_RESPONSE`
