package netsample

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/config"
)

func TestPhout(t *testing.T) {
	const fileName = "out.txt"

	tests := []struct {
		name      string
		resetConf func(cfg *PhoutConfig)
		reportCnt int
		want      string
	}{
		{
			name:      "no id by default",
			reportCnt: 2,
			want:      strings.Repeat(testSampleNoIDPhout+"\n", 2),
		},
		{
			name: "id option set",
			resetConf: func(cfg *PhoutConfig) {
				cfg.ID = true
			},
			reportCnt: 1,
			want:      testSamplePhout + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			conf := DefaultPhoutConfig()
			conf.Destination = fileName
			if tt.resetConf != nil {
				tt.resetConf(&conf)
			}
			ctx, cancel := context.WithCancel(context.Background())

			var err error
			testee, err := NewPhout(fs, conf)
			require.NoError(t, err)
			runErr := make(chan error)
			go func() {
				runErr <- testee.Run(ctx, core.AggregatorDeps{})
			}()

			for i := 0; i < tt.reportCnt; i++ {
				testee.Report(newTestSample())
			}
			cancel()
			err = <-runErr
			assert.NoError(t, err)

			data, err := afero.ReadFile(fs, fileName)
			require.NoError(t, err)

			assert.Equal(t, tt.want, string(data))
		})
	}
}

const (
	testSamplePhout     = "1484660999.002	tag1|tag2#42	333333	0	0	0	0	0	0	0	13	999"
	testSampleNoIDPhout = "1484660999.002	tag1|tag2	333333	0	0	0	0	0	0	0	13	999"
)

func newTestSample() *Sample {
	s := &Sample{}
	s.timeStamp = time.Unix(1484660999, 002*1000000)
	s.SetID(42)
	s.AddTag("tag1|tag2")
	s.setDuration(keyRTTMicro, time.Second/3)
	s.set(keyErrno, 13)
	s.set(keyProtoCode, ProtoCodeError)
	return s
}

// Голое число в YAML даёт НАНОСЕКУНДЫ: хук разбора (core/config/config.go:102) ловит только строки.
// Из-за этого flush-time: 1, написанный как «одна секунда», значит 1 нс — и его должен срезать валидатор.
func TestPhoutConfigFlushTimeValidation(t *testing.T) {
	decode := func(v interface{}) (PhoutConfig, error) {
		conf := DefaultPhoutConfig()
		conf.Destination = "out.txt"
		err := config.DecodeAndValidate(map[string]interface{}{"destination": "out.txt", "flush-time": v}, &conf)
		return conf, err
	}

	conf, err := decode("1s")
	require.NoError(t, err)
	require.Equal(t, time.Second, conf.FlushTime)

	// голое число = 1 нс
	_, err = decode(1)
	require.Error(t, err)

	// NewTicker на таком значении паникует
	_, err = decode("0s")
	require.Error(t, err)

	// флаш реже минуты бессмысленен: буфер всё равно сбросится по заполнению
	_, err = decode("5m")
	require.Error(t, err)
}

// Период флаша берётся из конфига, а не из зашитой секунды: с flush-time 20 мс данные должны
// оказаться в файле задолго до неё. Тест смотрит на файл, а не на поле конфига, иначе он не
// поймает возврат константы в NewTicker.
func TestPhoutFlushTimeUsed(t *testing.T) {
	const fileName = "flushed.txt"
	fs := afero.NewMemMapFs()
	conf := DefaultPhoutConfig()
	conf.Destination = fileName
	conf.FlushTime = 20 * time.Millisecond

	testee, err := NewPhout(fs, conf)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runErr := make(chan error, 1)
	go func() { runErr <- testee.Run(ctx, core.AggregatorDeps{}) }()

	testee.Report(newTestSample())

	// ждём флаша по тикеру, но заметно меньше секунды
	var data []byte
	require.Eventually(t, func() bool {
		data, _ = afero.ReadFile(fs, fileName)
		return len(data) > 0
	}, 400*time.Millisecond, 5*time.Millisecond, "сэмпл не сброшен в файл за 400 мс")
	require.Contains(t, string(data), testSampleNoIDPhout)

	cancel()
	<-runErr
}

// PhoutConfig могут собрать в коде мимо DefaultPhoutConfig — NewTicker(0) паникует.
func TestPhoutZeroFlushTimeFallsBackToSecond(t *testing.T) {
	fs := afero.NewMemMapFs()
	conf := DefaultPhoutConfig()
	conf.Destination = "zero.txt"
	conf.FlushTime = 0

	testee, err := NewPhout(fs, conf)
	require.NoError(t, err)
	require.Equal(t, time.Second, testee.(*phoutAggregator).config.FlushTime)
}
