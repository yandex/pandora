// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy

import (
	"io"
	logging "log"

	"github.com/SlyMarbo/spdy/common"
)

var log = common.GetLogger()
var debug = common.GetDebugLogger()

// SetLogger sets the package's error logger.
func SetLogger(l *logging.Logger) {
	common.SetLogger(l)
}

// SetLogOutput sets the output for the package's error logger.
func SetLogOutput(w io.Writer) {
	common.SetLogOutput(w)
}

// SetDebugLogger sets the package's debug info logger.
func SetDebugLogger(l *logging.Logger) {
	common.SetDebugLogger(l)
}

// SetDebugOutput sets the output for the package's debug info logger.
func SetDebugOutput(w io.Writer) {
	common.SetDebugOutput(w)
}

// EnableDebugOutput sets the output for the package's debug info logger to os.Stdout.
func EnableDebugOutput() {
	common.EnableDebugOutput()
}
