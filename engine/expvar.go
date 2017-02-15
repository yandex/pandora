package engine

// TODO: move as mach as possible to cli monitoring

import (
	"log"
	"time"

	"github.com/yandex/pandora/monitoring"
)

var (
	evRequests      = monitoring.NewCounter("engine_Requests")
	evResponses     = monitoring.NewCounter("engine_Responses")
	evUsersStarted  = monitoring.NewCounter("engine_UsersStarted")
	evUsersFinished = monitoring.NewCounter("engine_UsersFinished")
)

func init() {
	evReqPS := monitoring.NewCounter("engine_ReqPS")
	evResPS := monitoring.NewCounter("engine_ResPS")
	evActiveUsers := monitoring.NewCounter("engine_ActiveUsers")
	evActiveRequests := monitoring.NewCounter("engine_ActiveRequests")
	requests := evRequests.Get()
	responses := evResponses.Get()
	go func() {
		var requestsNew, responsesNew int64
		for range time.NewTicker(1 * time.Second).C {
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
