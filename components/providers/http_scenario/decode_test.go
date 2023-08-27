package httpscenario

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_spreadNames(t *testing.T) {
	tests := []struct {
		name      string
		input     []ScenarioConfig
		want      map[string]int
		wantTotal int
	}{
		{
			name:      "",
			input:     []ScenarioConfig{{Name: "a", Weight: 20}, {Name: "b", Weight: 30}, {Name: "c", Weight: 60}},
			want:      map[string]int{"a": 2, "b": 3, "c": 6},
			wantTotal: 11,
		},
		{
			name:      "",
			input:     []ScenarioConfig{{Name: "a", Weight: 100}, {Name: "b", Weight: 100}, {Name: "c", Weight: 100}},
			want:      map[string]int{"a": 1, "b": 1, "c": 1},
			wantTotal: 3,
		},
		{
			name:      "",
			input:     []ScenarioConfig{{Name: "a", Weight: 100}},
			want:      map[string]int{"a": 1},
			wantTotal: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total := spreadNames(tt.input)
			assert.Equalf(t, tt.want, got, "spreadNames(%v)", tt.input)
			assert.Equalf(t, tt.wantTotal, total, "spreadNames(%v)", tt.input)
		})
	}
}

func TestParseShootName(t *testing.T) {
	testCases := []struct {
		input    string
		wantName string
		wantCnt  int
		wantErr  bool
	}{
		{"shoot", "shoot", 1, false},
		{"shoot(5)", "shoot", 5, false},
		{"shoot(3,4,5)", "shoot", 3, false},
		{"shoot(5,6)", "shoot", 5, false},
		{"space test(7)", "space test", 7, false},
		{"symbol#(3)", "symbol#", 3, false},
		{"shoot(  9  )", "shoot", 9, false},
		{"shoot (6)", "shoot", 6, false},
		{"shoot()", "shoot", 1, false},
		{"shoot(abc)", "", 0, true},
		{"shoot(6", "", 0, true},
		{"shoot(6),", "", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			name, cnt, err := parseShootName(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantName, name, "Name does not match for input: %s", tc.input)
			assert.Equal(t, tc.wantCnt, cnt, "Count does not match for input: %s", tc.input)
		})
	}
}

func Test_convertScenarioToAmmo(t *testing.T) {
	req1 := RequestConfig{
		Method: "GET",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Name: "req1",
		URI:  "https://example.com/api/endpoint",
	}
	req2 := RequestConfig{
		Method: "POST",
		Headers: map[string]string{
			"Authorization": "Bearer abcdef",
		},
		Name: "req2",
		URI:  "https://example.com/api/another-endpoint",
	}

	reqRegistry := map[string]RequestConfig{
		"req1": req1,
		"req2": req2,
	}

	tests := []struct {
		name    string
		sc      ScenarioConfig
		want    *Ammo
		wantErr bool
	}{
		{
			name: "",
			sc: ScenarioConfig{
				Name:           "testScenario",
				Weight:         1,
				MinWaitingTime: 1000,
				Shoots: []string{
					"req1",
					"req2",
					"req2(2)",
					"sleep(500)",
				},
			},
			want: &Ammo{
				name:           "testScenario",
				minWaitingTime: time.Millisecond * 1000,
				Requests: []Request{
					convertConfigToRequestWithSleep(req1, 0),
					convertConfigToRequestWithSleep(req2, 0),
					convertConfigToRequestWithSleep(req2, 0),
					convertConfigToRequestWithSleep(req2, time.Millisecond*500),
				},
			},
			wantErr: false,
		},
		{
			name: "Scenario with unknown request",
			sc: ScenarioConfig{
				Name:           "unknownScenario",
				Weight:         1,
				MinWaitingTime: 1000,
				Shoots: []string{
					"unknownReq",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertScenarioToAmmo(tt.sc, reqRegistry)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			for i := range got.Requests {
				assert.NotNil(t, got.Requests[i].preprocessor)
				idx := got.Requests[i].preprocessor.iterator.Next("test")
				assert.Equal(t, i, idx) // this is a bit fragile, but it's ok for now
				got.Requests[i].preprocessor.iterator = nil
			}
			assert.Equalf(t, tt.want, got, "convertScenarioToAmmo(%v, %v)", tt.sc, reqRegistry)
		})
	}
}

func convertConfigToRequestWithSleep(req RequestConfig, sleep time.Duration) Request {
	res := convertConfigToRequest(req, nil)
	res.sleep = sleep
	return res
}
