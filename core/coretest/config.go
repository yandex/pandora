// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coretest

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/core/config"
)

func DecodeT(t *testing.T, data string, result interface{}) {
	conf := ParseYAML(t, data)
	err := config.Decode(conf, result)
	require.NoError(t, err)
}

func DecodeAndValidateT(t *testing.T, data string, result interface{}) {
	DecodeT(t, data, result)
	err := config.Validate(result)
	require.NoError(t, err)
}

func ParseYAML(t *testing.T, data string) map[string]interface{} {
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(data))
	require.NoError(t, err)
	return v.AllSettings()
}
