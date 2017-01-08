package aggregate

import (
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

const (
	keyRTTMicro     = iota
	keyConnectMicro // TODO: all for HTTP using httptrace and helper structs
	keySendMicro
	keyLatencyMicro
	keyReceiveMicro
	keyIntervalEventMicro // TODO: understand WTF is that mean and set it right.
	keyRequestBytes
	keyResponseBytes
	keyErrno
	keyProtoCode
	fieldsNum
)

func AcquireSample(tag string) *Sample {
	s := samplePool.Get().(*Sample)
	*s = Sample{
		timeStamp: time.Now(),
		tags:      tag,
	}
	return s
}

func ReleaseSample(s *Sample) { samplePool.Put(s) }

var samplePool = &sync.Pool{New: func() interface{} { return &Sample{} }}

type Sample struct {
	timeStamp time.Time
	tags      string
	fields    [fieldsNum]int
	err       error
}

func (s *Sample) Tags() string      { return s.tags }
func (s *Sample) AddTag(tag string) { s.tags += "|" + tag }

func (s *Sample) ProtoCode() int { return s.get(keyProtoCode) }
func (s *Sample) SetProtoCode(code int) {
	s.set(keyProtoCode, code)
	s.setRTT()
}

func (s *Sample) Err() error { return s.err }
func (s *Sample) SetErr(err error) {
	s.err = err
	s.set(keyErrno, getErrno(err))
	s.setRTT()
}

func (s *Sample) get(k int) int                      { return s.fields[k] }
func (s *Sample) set(k, v int)                       { s.fields[k] = v }
func (s *Sample) setDuration(k int, d time.Duration) { s.set(k, int(d.Nanoseconds()/1000)) }
func (s *Sample) setRTT() {
	if s.get(keyRTTMicro) == 0 {
		s.setDuration(keyRTTMicro, time.Since(s.timeStamp))
	}
}

func (s *Sample) String() string {
	return string(appendPhout(s, nil))
}

func getErrno(err error) int {
	// stackerr.Error and etc.
	type hasUnderlying interface {
		Underlying() error
	}
	for {
		typed, ok := err.(hasUnderlying)
		if !ok {
			break
		}
		err = typed.Underlying()
	}
	for {
		switch typed := err.(type) {
		case *net.OpError:
			err = typed.Err
		case *os.SyscallError:
			err = typed.Err
		case syscall.Errno:
			return int(typed)
		default:
			// Legacy default.
			return 999
		}
	}
}
