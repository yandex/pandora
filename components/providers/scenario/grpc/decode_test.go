package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gun "github.com/yandex/pandora/components/guns/grpc/scenario"
	"github.com/yandex/pandora/components/providers/scenario/config"
	"github.com/yandex/pandora/components/providers/scenario/grpc/postprocessor"
	"github.com/yandex/pandora/components/providers/scenario/grpc/preprocessor"
	"github.com/yandex/pandora/components/providers/scenario/vs"
)

func Test_decodeAmmo(t *testing.T) {
	storage := &vs.SourceStorage{}
	tests := []struct {
		name    string
		cfg     *config.AmmoConfig
		want    []*gun.Scenario
		wantErr bool
	}{
		{
			name: "full",
			cfg: &config.AmmoConfig{
				Scenarios: []config.ScenarioConfig{
					{
						Name:           "sc1",
						MinWaitingTime: 30,
						Weight:         1,
						Requests: []string{
							"req1(2, 100)",
							"req2",
							"sleep(200)",
						},
					},
					{
						Name:           "sc2",
						MinWaitingTime: 40,
						Weight:         2,
						Requests: []string{
							"req1(2, 300)",
							"sleep(100)",
							"req2",
							"sleep(400)",
						},
					},
				},
				Calls: []config.CallConfig{
					{
						Name:           "req1",
						Call:           "GET",
						Metadata:       map[string]string{"Content-Type": "application/json"},
						Tag:            "",
						Payload:        "http://localhost:8080/get",
						Preprocessors:  []gun.Preprocessor{&preprocessor.PreparePreprocessor{Mapping: map[string]string{"a": "b"}}},
						Postprocessors: []gun.Postprocessor{&postprocessor.AssertResponse{Payload: []string{"assert"}}},
					},
					{
						Name:           "req2",
						Call:           "POST",
						Metadata:       map[string]string{"Content-Type": "application/json", "Date": "2020-01-01"},
						Tag:            "",
						Payload:        "http://localhost:8080/post",
						Preprocessors:  []gun.Preprocessor{&preprocessor.PreparePreprocessor{Mapping: map[string]string{"c": "d"}}},
						Postprocessors: []gun.Postprocessor{&postprocessor.AssertResponse{Payload: []string{"assert2"}}},
					},
				},
			},
			want: []*gun.Scenario{
				{
					Calls: []gun.Call{
						{
							Call:           "GET",
							Metadata:       map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Name:           "req1",
							Payload:        []byte("http://localhost:8080/get"),
							Preprocessors:  []gun.Preprocessor{&preprocessor.PreparePreprocessor{Mapping: map[string]string{"a": "b"}}},
							Postprocessors: []gun.Postprocessor{&postprocessor.AssertResponse{Payload: []string{"assert"}}},
							Sleep:          100 * time.Millisecond,
						},
						{
							Call:           "GET",
							Metadata:       map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Name:           "req1",
							Payload:        []byte("http://localhost:8080/get"),
							Preprocessors:  []gun.Preprocessor{&preprocessor.PreparePreprocessor{Mapping: map[string]string{"a": "b"}}},
							Postprocessors: []gun.Postprocessor{&postprocessor.AssertResponse{Payload: []string{"assert"}}},
							Sleep:          100 * time.Millisecond,
						},
						{
							Call:           "POST",
							Metadata:       map[string]string{"Content-Type": "application/json", "Date": "2020-01-01"},
							Tag:            "",
							Name:           "req2",
							Payload:        []byte("http://localhost:8080/post"),
							Preprocessors:  []gun.Preprocessor{&preprocessor.PreparePreprocessor{Mapping: map[string]string{"c": "d"}}},
							Postprocessors: []gun.Postprocessor{&postprocessor.AssertResponse{Payload: []string{"assert2"}}},
							Sleep:          200 * time.Millisecond,
						},
					},
					Name:            "sc1",
					MinWaitingTime:  30 * time.Millisecond,
					VariableStorage: storage,
				},
				{
					Calls: []gun.Call{
						{
							Call:           "GET",
							Metadata:       map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Name:           "req1",
							Payload:        []byte("http://localhost:8080/get"),
							Preprocessors:  []gun.Preprocessor{&preprocessor.PreparePreprocessor{Mapping: map[string]string{"a": "b"}}},
							Postprocessors: []gun.Postprocessor{&postprocessor.AssertResponse{Payload: []string{"assert"}}},
							Sleep:          300 * time.Millisecond,
						},
						{
							Call:           "GET",
							Metadata:       map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Name:           "req1",
							Payload:        []byte("http://localhost:8080/get"),
							Preprocessors:  []gun.Preprocessor{&preprocessor.PreparePreprocessor{Mapping: map[string]string{"a": "b"}}},
							Postprocessors: []gun.Postprocessor{&postprocessor.AssertResponse{Payload: []string{"assert"}}},
							Sleep:          400 * time.Millisecond,
						},
						{
							Call:           "POST",
							Metadata:       map[string]string{"Content-Type": "application/json", "Date": "2020-01-01"},
							Tag:            "",
							Name:           "req2",
							Payload:        []byte("http://localhost:8080/post"),
							Preprocessors:  []gun.Preprocessor{&preprocessor.PreparePreprocessor{Mapping: map[string]string{"c": "d"}}},
							Postprocessors: []gun.Postprocessor{&postprocessor.AssertResponse{Payload: []string{"assert2"}}},
							Sleep:          400 * time.Millisecond,
						},
					},
					Name:            "sc2",
					MinWaitingTime:  40 * time.Millisecond,
					VariableStorage: storage,
				},
				{
					Calls: []gun.Call{
						{
							Call:           "GET",
							Metadata:       map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Name:           "req1",
							Payload:        []byte("http://localhost:8080/get"),
							Preprocessors:  []gun.Preprocessor{&preprocessor.PreparePreprocessor{Mapping: map[string]string{"a": "b"}}},
							Postprocessors: []gun.Postprocessor{&postprocessor.AssertResponse{Payload: []string{"assert"}}},
							Sleep:          300 * time.Millisecond,
						},
						{
							Call:           "GET",
							Metadata:       map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Name:           "req1",
							Payload:        []byte("http://localhost:8080/get"),
							Preprocessors:  []gun.Preprocessor{&preprocessor.PreparePreprocessor{Mapping: map[string]string{"a": "b"}}},
							Postprocessors: []gun.Postprocessor{&postprocessor.AssertResponse{Payload: []string{"assert"}}},
							Sleep:          400 * time.Millisecond,
						},
						{
							Call:           "POST",
							Metadata:       map[string]string{"Content-Type": "application/json", "Date": "2020-01-01"},
							Tag:            "",
							Name:           "req2",
							Payload:        []byte("http://localhost:8080/post"),
							Preprocessors:  []gun.Preprocessor{&preprocessor.PreparePreprocessor{Mapping: map[string]string{"c": "d"}}},
							Postprocessors: []gun.Postprocessor{&postprocessor.AssertResponse{Payload: []string{"assert2"}}},
							Sleep:          400 * time.Millisecond,
						},
					},
					Name:            "sc2",
					MinWaitingTime:  40 * time.Millisecond,
					VariableStorage: storage,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeAmmo(tt.cfg, storage)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			for _, s := range got {
				for _, c := range s.Calls {
					for _, p := range c.Preprocessors {
						if i, ok := p.(IteratorIniter); ok {
							i.InitIterator(nil)
						}
					}
				}
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
