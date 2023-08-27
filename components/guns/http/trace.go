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
	return time.Since(t.GotFirstResponseByte)
}

func (t *TraceTimings) GetConnectTime() time.Duration {
	return t.GotConnTime.Sub(t.GetConnTime)
}

func (t *TraceTimings) GetSendTime() time.Duration {
	return t.WroteRequestTime.Sub(t.GotConnTime)
}

func (t *TraceTimings) GetLatency() time.Duration {
	return t.GotFirstResponseByte.Sub(t.WroteRequestTime)
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
