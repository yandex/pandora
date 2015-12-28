package http

import (
	"fmt"
	"log"

	"github.com/yandex/pandora/aggregate"
)

type HttpSample struct {
	ts         float64 // Unix Timestamp in seconds
	rt         int     // response time in milliseconds
	StatusCode int     // protocol status code
	tag        string
	err        error
}

func (ds *HttpSample) PhoutSample() *aggregate.PhoutSample {
	var protoCode, netCode int
	if ds.err != nil {
		protoCode = 500
		netCode = 999
		log.Printf("Error code. %v\n", ds.err)
	} else {
		netCode = 0
		protoCode = ds.StatusCode
	}
	return &aggregate.PhoutSample{
		TS:            ds.ts,
		Tag:           ds.tag,
		RT:            ds.rt,
		Connect:       0,
		Send:          0,
		Latency:       0,
		Receive:       0,
		IntervalEvent: 0,
		Egress:        0,
		Igress:        0,
		NetCode:       netCode,
		ProtoCode:     protoCode,
	}
}

func (ds *HttpSample) String() string {
	return fmt.Sprintf("rt: %d [%d] %s", ds.rt, ds.StatusCode, ds.tag)
}
