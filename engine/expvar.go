package engine

import (
	"expvar"
	"strconv"
	"sync/atomic"
	"time"
)

type Counter struct {
	i int64
}

func (c *Counter) String() string {
	return strconv.FormatInt(atomic.LoadInt64(&c.i), 10)
}

func (c *Counter) Add(delta int64) {
	atomic.AddInt64(&c.i, delta)
}

func (c *Counter) Set(value int64) {
	atomic.StoreInt64(&c.i, value)
}

func (c *Counter) Get() int64 {
	return atomic.LoadInt64(&c.i)
}

func NewCounter(name string) *Counter {
	v := &Counter{}
	expvar.Publish(name, v)
	return v
}

var (
	evRequests      = NewCounter("engine_Requests")
	evResponses     = NewCounter("engine_Responses")
	evUsersStarted  = NewCounter("engine_UsersStarted")
	evUsersFinished = NewCounter("engine_UsersFinished")
)

func init() {
	evReqPS := NewCounter("engine_ReqPS")
	evResPS := NewCounter("engine_ResPS")
	evActiveUsers := NewCounter("engine_ActiveUsers")
	evActiveRequests := NewCounter("engine_ActiveRequests")
	requests := evRequests.Get()
	responses := evResponses.Get()
	go func() {
		var requestsNew, responsesNew int64
		for _ = range time.NewTicker(1 * time.Second).C {
			requestsNew = evRequests.Get()
			responsesNew = evResponses.Get()
			evActiveUsers.Set(evUsersStarted.Get() - evUsersFinished.Get())
			evActiveRequests.Set(requestsNew - responsesNew)
			evReqPS.Set(requestsNew - requests)
			evResPS.Set(responsesNew - responses)
			requests = requestsNew
			responses = responsesNew
		}
	}()
}
