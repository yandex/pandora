// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy

import (
	"errors"
	"sort"
)

// SPDY version of this implementation.
const DEFAULT_SPDY_VERSION = 3.1

// Version factors.
var supportedVersions = map[float64]struct{}{
	2:   struct{}{},
	3:   struct{}{},
	3.1: struct{}{},
}

const minVersion = 2
const maxVersion = 3.1

// SupportedVersions will return a slice of supported SPDY versions.
// The returned versions are sorted into order of most recent first.
func SupportedVersions() []float64 {
	s := make([]float64, 0, len(supportedVersions))
	for v, _ := range supportedVersions {
		s = append(s, v)
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(s)))
	return s
}

var npnStrings = map[float64]string{
	2:   "spdy/2",
	3:   "spdy/3",
	3.1: "spdy/3.1",
}

// npn returns the NPN version strings for the SPDY versions
// currently enabled, plus HTTP/1.1.
func npn() []string {
	v := SupportedVersions()
	s := make([]string, 0, len(v)+1)
	for _, v := range v {
		if str := npnStrings[float64(v)]; str != "" {
			s = append(s, str)
		}
	}
	s = append(s, "http/1.1")
	return s
}

// SupportedVersion determines if the provided SPDY version is
// supported by this instance of the library. This can be modified
// with EnableSpdyVersion and DisableSpdyVersion.
func SupportedVersion(v float64) bool {
	_, s := supportedVersions[v]
	return s
}

// EnableSpdyVersion can re-enable support for versions of SPDY
// that have been disabled by DisableSpdyVersion.
func EnableSpdyVersion(v float64) error {
	if v == 0 {
		return errors.New("Error: version 0 is invalid.")
	}
	if v < minVersion {
		return errors.New("Error: SPDY version too old.")
	}
	if v > maxVersion {
		return errors.New("Error: SPDY version too new.")
	}
	supportedVersions[v] = struct{}{}
	return nil
}

// DisableSpdyVersion can be used to disable support for the
// given SPDY version. This process can be undone by using
// EnableSpdyVersion.
func DisableSpdyVersion(v float64) error {
	if v == 0 {
		return errors.New("Error: version 0 is invalid.")
	}
	if v < minVersion {
		return errors.New("Error: SPDY version too old.")
	}
	if v > maxVersion {
		return errors.New("Error: SPDY version too new.")
	}
	delete(supportedVersions, v)
	return nil
}
