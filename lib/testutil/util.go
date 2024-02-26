// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package testutil

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func ReplaceGlobalLogger() *zap.Logger {
	log := NewLogger()
	zap.ReplaceGlobals(log)
	zap.RedirectStdLog(log)
	return log
}

func NewLogger() *zap.Logger {
	conf := zap.NewDevelopmentConfig()
	conf.OutputPaths = []string{"stdout"}
	conf.Level.SetLevel(zapcore.ErrorLevel)
	log, err := conf.Build(zap.AddCaller(), zap.AddStacktrace(zap.PanicLevel))
	if err != nil {
		zap.L().Fatal("Logger build failed", zap.Error(err))
	}
	return log
}

func NewNullLogger() *zap.Logger {
	c, _ := observer.New(zap.InfoLevel)
	return zap.New(c)
}
