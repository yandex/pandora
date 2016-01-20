package engine

import (
	"expvar"
	"log"
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
			rps := responsesNew - responses
			reqps := requestsNew - requests
			activeUsers := evUsersStarted.Get() - evUsersFinished.Get()
			activeRequests := requestsNew - responsesNew
			log.Printf(
				"[ENGINE] %d resp/s; %d req/s; %d users; %d active\n",
				rps, reqps, activeUsers, activeRequests)

			requests = requestsNew
			responses = responsesNew

			evActiveUsers.Set(activeUsers)
			evActiveRequests.Set(activeRequests)
			evReqPS.Set(reqps)
			evResPS.Set(rps)
		}
	}()
}
