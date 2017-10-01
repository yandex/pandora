// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package testutil

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func RunSuite(t *testing.T, description string) {
	format.UseStringerRepresentation = true
	ReplaceGlobalLogger()
	RegisterFailHandler(Fail)
	RunSpecs(t, description)
}

func ReplaceGlobalLogger() *zap.Logger {
	log := NewLogger()
	zap.ReplaceGlobals(log)
	zap.RedirectStdLog(log)
	return log
}

func NewLogger() *zap.Logger {
	conf := zap.NewDevelopmentConfig()
	enc := zapcore.NewConsoleEncoder(conf.EncoderConfig)
	core := zapcore.NewCore(enc, zapcore.AddSync(GinkgoWriter), zap.DebugLevel)
	log := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.DPanicLevel))
	return log
}

type Mock interface {
	AssertExpectations(t mock.TestingT) bool
	AssertNotCalled(t mock.TestingT, methodName string, arguments ...interface{}) bool
}

func AssertExpectations(mocks ...Mock) {
	for _, m := range mocks {
		m.AssertExpectations(GinkgoT(1))
	}
}

func AssertNotCalled(mock Mock, methodName string) {
	mock.AssertNotCalled(GinkgoT(1), methodName)
}

func ParseYAML(data string) map[string]interface{} {
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(strings.NewReader(data))
	Expect(err).NotTo(HaveOccurred())
	return v.AllSettings()
}
