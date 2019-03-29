// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregator

import (
	"bufio"
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/yandex/pandora/lib/ioutil2"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/coreutil"
)

type JSONLineAggregatorConfig struct {
	EncoderAggregatorConfig `config:",squash"`
	JSONLineEncoderConfig   `config:",squash"`
}

type JSONLineEncoderConfig struct {
	JSONIterConfig            `config:",squash"`
	coreutil.BufferSizeConfig `config:",squash"`
}

// JSONIterConfig is subset of jsoniter.Config that may be useful to configure.
type JSONIterConfig struct {
	// MarshalFloatWith6Digits makes float marshalling faster.
	MarshalFloatWith6Digits bool `config:"marshal-float-with-6-digits"`
	// SortMapKeys useful, when sample contains map object, and you want to see them in same order.
	SortMapKeys bool `config:"sort-map-keys"`
}

func DefaultJSONLinesAggregatorConfig() JSONLineAggregatorConfig {
	return JSONLineAggregatorConfig{
		EncoderAggregatorConfig: DefaultEncoderAggregatorConfig(),
	}
}

// Aggregates samples in JSON Lines format: each output line is a Valid JSON Value of one sample.
// See http://jsonlines.org/ for details.
func NewJSONLinesAggregator(conf JSONLineAggregatorConfig) core.Aggregator {
	var newEncoder NewSampleEncoder = func(w io.Writer, onFlush func()) SampleEncoder {
		w = ioutil2.NewCallbackWriter(w, onFlush)
		return NewJSONEncoder(w, conf.JSONLineEncoderConfig)
	}
	return NewEncoderAggregator(newEncoder, conf.EncoderAggregatorConfig)
}

func NewJSONEncoder(w io.Writer, conf JSONLineEncoderConfig) SampleEncoder {
	var apiConfig jsoniter.Config
	config.Map(&apiConfig, conf.JSONIterConfig)
	api := apiConfig.Froze()
	// NOTE(skipor): internal buffering is not working really. Don't know why
	// OPTIMIZE(skipor): don't wrap into buffer, if already ioutil2.ByteWriter
	buf := bufio.NewWriterSize(w, conf.BufferSizeOrDefault())
	stream := jsoniter.NewStream(api, buf, conf.BufferSizeOrDefault())
	return &jsonEncoder{stream, buf}
}

type jsonEncoder struct {
	*jsoniter.Stream
	buf *bufio.Writer
}

func (e *jsonEncoder) Encode(s core.Sample) error {
	e.WriteVal(s)
	e.WriteRaw("\n")
	return e.Error
}

func (e *jsonEncoder) Flush() error {
	err := e.Stream.Flush()
	e.buf.Flush()
	return err
}
