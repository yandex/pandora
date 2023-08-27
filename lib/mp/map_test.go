package mp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMapValue(t *testing.T) {
	tests := []struct {
		name    string
		reqMap  map[string]any
		v       string
		want    any
		wantErr bool
	}{
		{
			name: "",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:       "source.items[0].id",
			want:    "1",
			wantErr: false,
		},
		{
			name: "",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:       "source.items[1]",
			want:    map[string]any{"id": "2"},
			wantErr: false,
		},
		{
			name: "",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:       "source.items[1].title",
			want:    nil,
			wantErr: true,
		},
		{
			name: "",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v: "source.items",
			want: []map[string]any{
				{"id": "1"},
				{"id": "2"},
			},
			wantErr: false,
		},
		{
			name: "",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]string{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:       "source.items[0].id",
			want:    "1",
			wantErr: false,
		},
		{
			name: "",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []any{11, 22, 33},
				},
			},
			v:       "source.items[0]",
			want:    11,
			wantErr: false,
		},
		{
			name: "",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []any{11, 22, 33},
				},
			},
			v:       "source.items[0].id",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := NewNextIterator(0)
			got, err := GetMapValue(tt.reqMap, tt.v, iter)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equalf(t, tt.want, got, "getValue(%v, %v)", tt.reqMap, tt.v)
		})
	}
}

func Test_getValue_iterators(t *testing.T) {
	iter := NewNextIterator(0)

	reqMap := map[string]any{
		"source": map[string]any{
			"items": []any{11, 22, 33},
			"list":  []string{"11", "22", "33"},
		},
	}
	var got any

	got, _ = GetMapValue(reqMap, "source.list[next]", iter)
	assert.Equal(t, "11", got)
	got, _ = GetMapValue(reqMap, "source.list[next]", iter)
	assert.Equal(t, "22", got)
	got, _ = GetMapValue(reqMap, "source.list[next]", iter)
	assert.Equal(t, "33", got)
	got, _ = GetMapValue(reqMap, "source.list[next]", iter)
	assert.Equal(t, "11", got)
	got, _ = GetMapValue(reqMap, "source.list[next]", iter)
	assert.Equal(t, "22", got)
	got, _ = GetMapValue(reqMap, "source.list[next]", iter)
	assert.Equal(t, "33", got)

	got, _ = GetMapValue(reqMap, "source.list[last]", iter)
	assert.Equal(t, "33", got)
	got, _ = GetMapValue(reqMap, "source.list[last]", iter)
	assert.Equal(t, "33", got)
	got, _ = GetMapValue(reqMap, "source.list[-2]", iter)
	assert.Equal(t, "22", got)

	got, _ = GetMapValue(reqMap, "source.items[rand]", iter)
	assert.Equal(t, 11, got)
	got, _ = GetMapValue(reqMap, "source.items[rand]", iter)
	assert.Equal(t, 11, got)
	got, _ = GetMapValue(reqMap, "source.items[rand]", iter)
	assert.Equal(t, 22, got)
	got, _ = GetMapValue(reqMap, "source.items[rand]", iter)
	assert.Equal(t, 22, got)
	got, _ = GetMapValue(reqMap, "source.items[rand]", iter)
	assert.Equal(t, 33, got)
	got, _ = GetMapValue(reqMap, "source.items[rand]", iter)
	assert.Equal(t, 22, got)

	got, _ = GetMapValue(reqMap, "source.items[next]", iter)
	assert.Equal(t, 11, got)
	got, _ = GetMapValue(reqMap, "source.items[next]", iter)
	assert.Equal(t, 22, got)
	got, _ = GetMapValue(reqMap, "source.items[next]", iter)
	assert.Equal(t, 33, got)
	got, _ = GetMapValue(reqMap, "source.items[next]", iter)
	assert.Equal(t, 11, got)
	got, _ = GetMapValue(reqMap, "source.items[next]", iter)
	assert.Equal(t, 22, got)

}
