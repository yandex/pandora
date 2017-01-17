// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"io"
	"io/ioutil"
	logging "log"
	"os"
)

type Logger struct {
	*logging.Logger
}

var log = &Logger{logging.New(os.Stderr, "(spdy) ", logging.LstdFlags|logging.Lshortfile)}
var debug = &Logger{logging.New(ioutil.Discard, "(spdy debug) ", logging.LstdFlags)}
var VerboseLogging = false

func GetLogger() *Logger {
	return log
}

func GetDebugLogger() *Logger {
	return debug
}

// SetLogger sets the package's error logger.
func SetLogger(l *logging.Logger) {
	log.Logger = l
}

// SetLogOutput sets the output for the package's error logger.
func SetLogOutput(w io.Writer) {
	log.Logger = logging.New(w, "(spdy) ", logging.LstdFlags|logging.Lshortfile)
}

// SetDebugLogger sets the package's debug info logger.
func SetDebugLogger(l *logging.Logger) {
	debug.Logger = l
}

// SetDebugOutput sets the output for the package's debug info logger.
func SetDebugOutput(w io.Writer) {
	debug.Logger = logging.New(w, "(spdy debug) ", logging.LstdFlags)
}

// EnableDebugOutput sets the output for the package's debug info logger to os.Stdout.
func EnableDebugOutput() {
	SetDebugOutput(os.Stdout)
}
