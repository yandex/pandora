// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregator

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/datasink"
)

type jsonTestData struct {
	String string `json:"string"`
	Int    int    `json:"int"`
}

func TestNewJSONLinesAggregator(t *testing.T) {
	samples := []jsonTestData{
		{"A", 1},
		{"B", 2},
		{"C", 3},
	}

	conf := DefaultJSONLinesAggregatorConfig()
	sink := &datasink.Buffer{}
	conf.Sink = sink
	testee := NewJSONLinesAggregator(conf)
	ctx, cancel := context.WithCancel(context.Background())

	runErr := make(chan error)
	go func() {
		runErr <- testee.Run(ctx, core.AggregatorDeps{Log: zap.L()})
	}()

	for _, sample := range samples {
		testee.Report(sample)
	}
	cancel()
	err := <-runErr
	require.NoError(t, err)

	for _, expected := range samples {
		var actual jsonTestData
		line, err := sink.ReadBytes('\n')
		require.NoError(t, err)
		err = json.Unmarshal(line, &actual)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	}

	assert.Zero(t, sink.Len())
}
