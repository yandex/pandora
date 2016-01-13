package aggregate

import (
	"fmt"
	"strconv"
	"sync"
)

const (
	phoutDelimiter = '\t'
	phoutNewLine   = '\n'
)

var samplePool = &sync.Pool{New: func() interface{} { return &Sample{} }}

type Sample struct {
	TS            float64
	Tag           string
	RT            int
	Connect       int
	Send          int
	Latency       int
	Receive       int
	IntervalEvent int
	Egress        int
	Igress        int
	NetCode       int
	ProtoCode     int
	Err           error
}

func AcquireSample() *Sample {
	return samplePool.Get().(*Sample)
}

func ReleaseSample(s *Sample) {
	samplePool.Put(s)
}

func (ps *Sample) String() string {
	return fmt.Sprintf(
		"%.3f\t%s\t%d\t"+
			"%d\t%d\t"+
			"%d\t%d\t"+
			"%d\t"+
			"%d\t%d\t"+
			"%d\t%d",
		ps.TS, ps.Tag, ps.RT,
		ps.Connect, ps.Send,
		ps.Latency, ps.Receive,
		ps.IntervalEvent,
		ps.Egress, ps.Igress,
		ps.NetCode, ps.ProtoCode,
	)
}

func (ps *Sample) AppendToPhout(dst []byte) []byte {
	dst = strconv.AppendFloat(dst, ps.TS, 'f', 3, 64)
	dst = append(dst, phoutDelimiter)
	dst = append(dst, ps.Tag...)
	dst = append(dst, phoutDelimiter)
	dst = strconv.AppendInt(dst, int64(ps.RT), 10)
	dst = append(dst, phoutDelimiter)
	dst = strconv.AppendInt(dst, int64(ps.Connect), 10)
	dst = append(dst, phoutDelimiter)
	dst = strconv.AppendInt(dst, int64(ps.Send), 10)
	dst = append(dst, phoutDelimiter)
	dst = strconv.AppendInt(dst, int64(ps.Latency), 10)
	dst = append(dst, phoutDelimiter)
	dst = strconv.AppendInt(dst, int64(ps.Receive), 10)
	dst = append(dst, phoutDelimiter)
	dst = strconv.AppendInt(dst, int64(ps.IntervalEvent), 10)
	dst = append(dst, phoutDelimiter)
	dst = strconv.AppendInt(dst, int64(ps.Egress), 10)
	dst = append(dst, phoutDelimiter)
	dst = strconv.AppendInt(dst, int64(ps.Igress), 10)
	dst = append(dst, phoutDelimiter)
	dst = strconv.AppendInt(dst, int64(ps.NetCode), 10)
	dst = append(dst, phoutDelimiter)
	dst = strconv.AppendInt(dst, int64(ps.ProtoCode), 10)
	dst = append(dst, phoutNewLine)
	return dst
}
