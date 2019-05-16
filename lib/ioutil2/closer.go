// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package ioutil2

// NopCloser may be embedded to any struct to implement io.Closer doing nothing on closer.
type NopCloser struct{}

func (NopCloser) Close() error { return nil }
