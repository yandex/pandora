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
