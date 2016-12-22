// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregate

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/facebookgo/stackerr"
	"github.com/stretchr/testify/assert"
)

func TestSampleBehaviour(t *testing.T) {
	const tag = "test"
	const tag2 = "test2"
	sample := AcquireSample(tag)
	sample.AddTag(tag2)
	const sleep = time.Millisecond
	time.Sleep(sleep)
	sample.SetErr(syscall.EINVAL)
	rtt := time.Duration(sample.get(keyRTTMicro)) * time.Microsecond
	assert.NotZero(t, rtt)
	assert.True(t, rtt <= time.Since(sample.timeStamp), "expected: %v; actual: %v", rtt, time.Since(sample.timeStamp))
	assert.True(t, sleep <= rtt)
	sample.SetProtoCode(http.StatusBadRequest)
	expected := fmt.Sprintf("%v.%3.f\t%s|%s\t%v\t0\t0\t0\t0\t0\t0\t0\t%v\t%v",
		sample.timeStamp.Unix(),
		float32((sample.timeStamp.UnixNano()/1e6)%1000),
		tag, tag2,
		sample.get(keyRTTMicro),
		int(syscall.EINVAL), http.StatusBadRequest,
	)
	assert.Equal(t, expected, sample.String())
}

func TestGetErrno(t *testing.T) {
	var err error = syscall.EINVAL
	err = &os.SyscallError{Err: err}
	err = &net.OpError{Err: err}
	err = stackerr.Wrap(err)
	assert.NotZero(t, getErrno(err))
	assert.Equal(t, int(syscall.EINVAL), getErrno(err))
}
