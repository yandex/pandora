package phttp

import (
	"net/http/httptrace"
	"time"
)

type TraceTimings struct {
	GotConnTime          time.Time
	GetConnTime          time.Time
	DNSStartTime         time.Time
	DNSDoneTime          time.Time
	ConnectDoneTime      time.Time
	ConnectStartTime     time.Time
	WroteRequestTime     time.Time
	GotFirstResponseByte time.Time
}

func (t *TraceTimings) GetReceiveTime() time.Duration {
	return between(t.GotFirstResponseByte, time.Now())
}

func (t *TraceTimings) GetConnectTime() time.Duration {
	return between(t.GetConnTime, t.GotConnTime)
}

func (t *TraceTimings) GetSendTime() time.Duration {
	return between(t.GotConnTime, t.WroteRequestTime)
}

func (t *TraceTimings) GetLatency() time.Duration {
	return between(t.WroteRequestTime, t.GotFirstResponseByte)
}

// Событие, которое не наступило (запрос упал раньше), даёт 0: Sub от нулевого time.Time насыщается до ±2^63 нс.
func between(from, to time.Time) time.Duration {
	if from.IsZero() || to.IsZero() {
		return 0
	}
	return to.Sub(from)
}

func CreateHTTPTrace() (*httptrace.ClientTrace, *TraceTimings) {
	timings := &TraceTimings{}
	tracer := &httptrace.ClientTrace{
		GetConn: func(_ string) {
			timings.GetConnTime = time.Now()
		},
		GotConn: func(_ httptrace.GotConnInfo) {
			timings.GotConnTime = time.Now()
		},
		DNSStart: func(_ httptrace.DNSStartInfo) {
			timings.DNSStartTime = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			timings.DNSDoneTime = time.Now()
		},
		ConnectStart: func(network, addr string) {
			timings.ConnectStartTime = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			timings.ConnectDoneTime = time.Now()
		},
		WroteRequest: func(wr httptrace.WroteRequestInfo) {
			timings.WroteRequestTime = time.Now()
		},
		GotFirstResponseByte: func() {
			timings.GotFirstResponseByte = time.Now()
		},
	}

	return tracer, timings
}
