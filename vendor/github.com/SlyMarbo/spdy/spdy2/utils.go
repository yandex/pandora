// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy2

import (
	"github.com/SlyMarbo/spdy/common"
)

// defaultServerSettings are used in initialising the connection.
// It takes the max concurrent streams.
func defaultServerSettings(m uint32) common.Settings {
	return common.Settings{
		common.SETTINGS_MAX_CONCURRENT_STREAMS: &common.Setting{
			Flags: common.FLAG_SETTINGS_PERSIST_VALUE,
			ID:    common.SETTINGS_MAX_CONCURRENT_STREAMS,
			Value: m,
		},
	}
}

// defaultClientSettings are used in initialising the connection.
// It takes the max concurrent streams.
func defaultClientSettings(m uint32) common.Settings {
	return common.Settings{
		common.SETTINGS_MAX_CONCURRENT_STREAMS: &common.Setting{
			ID:    common.SETTINGS_MAX_CONCURRENT_STREAMS,
			Value: m,
		},
	}
}
