// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package testutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.uber.org/atomic"
)

type TestingT interface {
	mock.TestingT
}

type flakyT struct {
	t      *testing.T
	failed atomic.Bool
}

var _ TestingT = &flakyT{}

func (ff *flakyT) Logf(format string, args ...interface{}) {
	getHelper(ff.t).Helper()
	ff.Logf(format, args...)
}

func (ff *flakyT) Errorf(format string, args ...interface{}) {
	getHelper(ff.t).Helper()
	ff.t.Logf(format, args...)
	ff.failed.Store(true)
}

func (ff *flakyT) FailNow() {
	// WARN: that may not work in complex case.
	panic("Failing now!")
}

func RunFlaky(t *testing.T, test func(t TestingT)) {
	const retries = 5
	var passed bool
	for i := 1; i <= retries && !passed; i++ {
		t.Run(fmt.Sprintf("Retry_%v", i), func(t *testing.T) {
			if i == retries {
				t.Log("Last retry.")
				test(t)
				return
			}
			ff := &flakyT{t: t}

			var hasPanic bool
			func() {
				defer func() {
					hasPanic = recover() != nil
				}()
				test(ff)
			}()

			passed = !ff.failed.Load() && !hasPanic
			if passed {
				t.Log("Passed!")
			} else {
				t.Log("Failed! Retrying.")
			}
		})
	}
}

// getHelper allows to call t.Helper() without breaking compatibility with go version < 1.9
func getHelper(t TestingT) helper {
	var tInterface interface{} = t
	if h, ok := tInterface.(helper); ok {
		return h
	}
	return nopHelper{}
}

type nopHelper struct{}

func (nopHelper) Helper() {}

type helper interface {
	Helper()
}
