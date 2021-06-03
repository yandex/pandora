// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package netsample

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/facebookgo/stackerr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestSampleBehaviour(t *testing.T) {
	const tag = "test"
	const tag2 = "test2"
	const id = 42
	sample := Acquire(tag)
	sample.AddTag(tag2)
	sample.SetID(id)
	const sleep = time.Millisecond
	time.Sleep(sleep)
	sample.SetErr(syscall.EINVAL)
	rtt := time.Duration(sample.get(keyRTTMicro)) * time.Microsecond
	assert.NotZero(t, rtt)
	assert.True(t, rtt <= time.Since(sample.timeStamp), "expected: %v; actual: %v", rtt, time.Since(sample.timeStamp))
	assert.True(t, sleep <= rtt)
	sample.SetProtoCode(http.StatusBadRequest)
	expectedTimeStamp := fmt.Sprintf("%v.%3.f",
		sample.timeStamp.Unix(),
		float32((sample.timeStamp.UnixNano()/1e6)%1000))
	// 1484660999.  2 -> 1484660999.002
	expectedTimeStamp = strings.Replace(expectedTimeStamp, " ", "0", -1)

	expected := fmt.Sprintf("%s\t%s|%s#%v\t%v\t0\t0\t0\t0\t0\t0\t0\t%v\t%v",
		expectedTimeStamp,
		tag, tag2, id,
		sample.get(keyRTTMicro),
		int(syscall.EINVAL), http.StatusBadRequest,
	)
	assert.Equal(t, expected, sample.String())
}

func TestCustomSets(t *testing.T) {
	const tag = "UserDefine"
	s := Acquire(tag)

	userDuration := 100 * time.Millisecond
	s.SetUserDuration(userDuration)

	s.SetUserProto(0)
	s.SetUserNet(110)

	latency := 200 * time.Millisecond
	s.SetLatency(latency)

	reqBytes := 4
	s.SetRequestBytes(reqBytes)

	respBytes := 8
	s.SetResponceBytes(respBytes)

	expectedTimeStamp := fmt.Sprintf("%v.%3.f",
		s.timeStamp.Unix(),
		float32((s.timeStamp.UnixNano()/1e6)%1000))
	expectedTimeStamp = strings.Replace(expectedTimeStamp, " ", "0", -1)
	expected := fmt.Sprintf("%s\t%s#0\t%v\t0\t0\t%v\t0\t0\t%v\t%v\t%v\t%v",
		expectedTimeStamp,
		tag,
		int(userDuration.Nanoseconds()/1000), // keyRTTMicro
		int(latency.Nanoseconds()/1000),      // keyLatencyMicro
		reqBytes,                             // keyRequestBytes
		respBytes,                            // keyResponseBytes
		110,
		0,
	)
	assert.Equal(t, s.String(), expected)
}

func TestGetErrno(t *testing.T) {
	var err error = syscall.EINVAL
	err = &os.SyscallError{Err: err}
	err = &net.OpError{Err: err}
	err = errors.WithStack(err)
	err = stackerr.Wrap(err)
	assert.NotZero(t, getErrno(err))
	assert.Equal(t, int(syscall.EINVAL), getErrno(err))
}

// TODO (skipor): test getErrno on some real net error from stdlib.

func BenchmarkAppendTimestamp(b *testing.B) {
	dst := make([]byte, 0, 512)
	ts := time.Now()
	b.Run("DotInsert", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			appendTimestamp(ts, dst)
			dst = dst[:0]
		}
	})
	b.Run("UseMod", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			dst = strconv.AppendInt(dst, ts.Unix(), 10)
			dst = append(dst, '.')
			dst = strconv.AppendInt(dst, (ts.UnixNano()/1e3)%1e3, 10)
			dst = dst[:0]
		}
	})
	b.Run("NoDot", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			dst = strconv.AppendInt(dst, ts.UnixNano()/1e3, 10)
			dst = dst[:0]
		}
	})
}
