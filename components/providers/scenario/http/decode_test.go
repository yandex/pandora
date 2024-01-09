package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gun "github.com/yandex/pandora/components/guns/http_scenario"
	"github.com/yandex/pandora/components/providers/scenario/config"
	"github.com/yandex/pandora/components/providers/scenario/http/postprocessor"
	"github.com/yandex/pandora/components/providers/scenario/http/preprocessor"
	"github.com/yandex/pandora/components/providers/scenario/http/templater"
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
				Requests: []config.RequestConfig{
					{
						Name:           "req1",
						Method:         "GET",
						Headers:        map[string]string{"Content-Type": "application/json"},
						Tag:            "",
						Body:           nil,
						URI:            "http://localhost:8080/get",
						Preprocessor:   &preprocessor.Preprocessor{Mapping: map[string]string{"a": "b"}},
						Postprocessors: []gun.Postprocessor{&postprocessor.VarHeaderPostprocessor{}, &postprocessor.VarJsonpathPostprocessor{}},
						Templater:      templater.NewHTMLTemplater(),
					},
					{
						Name:           "req2",
						Method:         "POST",
						Headers:        map[string]string{"Content-Type": "application/json", "Date": "2020-01-01"},
						Tag:            "",
						Body:           nil,
						URI:            "http://localhost:8080/post",
						Preprocessor:   &preprocessor.Preprocessor{Mapping: map[string]string{"c": "d"}},
						Postprocessors: []gun.Postprocessor{&postprocessor.VarXpathPostprocessor{}},
						Templater:      nil,
					},
				},
			},
			want: []*gun.Scenario{
				{
					Requests: []gun.Request{
						{
							Method:         "GET",
							Headers:        map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Body:           nil,
							Name:           "req1",
							URI:            "http://localhost:8080/get",
							Preprocessor:   &preprocessor.Preprocessor{Mapping: map[string]string{"a": "b"}},
							Postprocessors: []gun.Postprocessor{&postprocessor.VarHeaderPostprocessor{}, &postprocessor.VarJsonpathPostprocessor{}},
							Templater:      templater.NewHTMLTemplater(),
							Sleep:          100 * time.Millisecond,
						},
						{
							Method:         "GET",
							Headers:        map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Body:           nil,
							Name:           "req1",
							URI:            "http://localhost:8080/get",
							Preprocessor:   &preprocessor.Preprocessor{Mapping: map[string]string{"a": "b"}},
							Postprocessors: []gun.Postprocessor{&postprocessor.VarHeaderPostprocessor{}, &postprocessor.VarJsonpathPostprocessor{}},
							Templater:      templater.NewHTMLTemplater(),
							Sleep:          100 * time.Millisecond,
						},
						{
							Method:         "POST",
							Headers:        map[string]string{"Content-Type": "application/json", "Date": "2020-01-01"},
							Tag:            "",
							Body:           nil,
							Name:           "req2",
							URI:            "http://localhost:8080/post",
							Preprocessor:   &preprocessor.Preprocessor{Mapping: map[string]string{"c": "d"}},
							Postprocessors: []gun.Postprocessor{&postprocessor.VarXpathPostprocessor{}},
							Templater:      templater.NewTextTemplater(),
							Sleep:          200 * time.Millisecond,
						},
					},
					ID:              0,
					Name:            "sc1",
					MinWaitingTime:  30 * time.Millisecond,
					VariableStorage: storage,
				},
				{
					Requests: []gun.Request{
						{
							Method:         "GET",
							Headers:        map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Body:           nil,
							Name:           "req1",
							URI:            "http://localhost:8080/get",
							Preprocessor:   &preprocessor.Preprocessor{Mapping: map[string]string{"a": "b"}},
							Postprocessors: []gun.Postprocessor{&postprocessor.VarHeaderPostprocessor{}, &postprocessor.VarJsonpathPostprocessor{}},
							Templater:      templater.NewHTMLTemplater(),
							Sleep:          300 * time.Millisecond,
						},
						{
							Method:         "GET",
							Headers:        map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Body:           nil,
							Name:           "req1",
							URI:            "http://localhost:8080/get",
							Preprocessor:   &preprocessor.Preprocessor{Mapping: map[string]string{"a": "b"}},
							Postprocessors: []gun.Postprocessor{&postprocessor.VarHeaderPostprocessor{}, &postprocessor.VarJsonpathPostprocessor{}},
							Templater:      templater.NewHTMLTemplater(),
							Sleep:          400 * time.Millisecond,
						},
						{
							Method:         "POST",
							Headers:        map[string]string{"Content-Type": "application/json", "Date": "2020-01-01"},
							Tag:            "",
							Body:           nil,
							Name:           "req2",
							URI:            "http://localhost:8080/post",
							Preprocessor:   &preprocessor.Preprocessor{Mapping: map[string]string{"c": "d"}},
							Postprocessors: []gun.Postprocessor{&postprocessor.VarXpathPostprocessor{}},
							Templater:      templater.NewTextTemplater(),
							Sleep:          400 * time.Millisecond,
						},
					},
					ID:              0,
					Name:            "sc2",
					MinWaitingTime:  40 * time.Millisecond,
					VariableStorage: storage,
				},
				{
					Requests: []gun.Request{
						{
							Method:         "GET",
							Headers:        map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Body:           nil,
							Name:           "req1",
							URI:            "http://localhost:8080/get",
							Preprocessor:   &preprocessor.Preprocessor{Mapping: map[string]string{"a": "b"}},
							Postprocessors: []gun.Postprocessor{&postprocessor.VarHeaderPostprocessor{}, &postprocessor.VarJsonpathPostprocessor{}},
							Templater:      templater.NewHTMLTemplater(),
							Sleep:          300 * time.Millisecond,
						},
						{
							Method:         "GET",
							Headers:        map[string]string{"Content-Type": "application/json"},
							Tag:            "",
							Body:           nil,
							Name:           "req1",
							URI:            "http://localhost:8080/get",
							Preprocessor:   &preprocessor.Preprocessor{Mapping: map[string]string{"a": "b"}},
							Postprocessors: []gun.Postprocessor{&postprocessor.VarHeaderPostprocessor{}, &postprocessor.VarJsonpathPostprocessor{}},
							Templater:      templater.NewHTMLTemplater(),
							Sleep:          400 * time.Millisecond,
						},
						{
							Method:         "POST",
							Headers:        map[string]string{"Content-Type": "application/json", "Date": "2020-01-01"},
							Tag:            "",
							Body:           nil,
							Name:           "req2",
							URI:            "http://localhost:8080/post",
							Preprocessor:   &preprocessor.Preprocessor{Mapping: map[string]string{"c": "d"}},
							Postprocessors: []gun.Postprocessor{&postprocessor.VarXpathPostprocessor{}},
							Templater:      templater.NewTextTemplater(),
							Sleep:          400 * time.Millisecond,
						},
					},
					ID:              0,
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
				for _, r := range s.Requests {
					if p, ok := r.Preprocessor.(IteratorIniter); ok {
						p.InitIterator(nil)
					}
				}
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
