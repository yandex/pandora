package mp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMapValue(t *testing.T) {
	var cases = []struct {
		name    string
		reqMap  map[string]any
		v       string
		want    any
		wantErr string
	}{
		{
			name: "valid index for nested map with array map",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:    "source.items[0].id",
			want: "1",
		},
		{
			name: "invalid index for nested map with array map",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:       "source.items[asd].id",
			wantErr: "failed to calc index for []map[string]interface {}; err: index should be integer or one of [next, rand, last], but got `asd`",
		},
		{
			name: "should slice type error ",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:       "source.items[0].id[0]",
			wantErr: "cant extract value path=`id`,segment=`source.items[0].id[0]`,err=invalid type of value `1`, string",
		},
		{
			name: "not last segment",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:       "source.items[0].id.field",
			wantErr: "not last segment id in path source.items[0].id.field",
		},
		{
			name: "segment data not found",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:       "source.data[0].id",
			want:    "1",
			wantErr: "segment data not found in path source.data[0].id",
		},
		{
			name: "extract map from array / element in array / valid index",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:    "source.items[1]",
			want: map[string]any{"id": "2"},
		},
		{
			name: "retrieve non existent key / error when searching for missing key",
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
			wantErr: "segment title not found in path source.items[1].title",
		},
		{
			name: "access items in nested map / valid path / return items list",
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
		},
		{
			name: "valid value for map[string]string in map[string]any",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]string{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:    "source.items[0].id",
			want: "1",
		},
		{
			name: "invalid index for map[string]string in map[string]any",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]string{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:       "source.items[asd].id",
			want:    "1",
			wantErr: "failed to calc index for []map[string]string; err: index should be integer or one of [next, rand, last], but got `asd`",
		},
		{
			name: "duplicate test case / same as / test case5",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []map[string]any{
						{"id": "1"},
						{"id": "2"},
					},
				},
			},
			v:    "source.items[0].id",
			want: "1",
		},
		{
			name: "valid key for []any in map[string]any",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []any{
						map[string]any{
							"id":   "1",
							"name": "name1",
						},
						map[string]any{
							"id":   "2",
							"name": "name2",
						},
					},
				},
			},
			v:    "source.items[next].name",
			want: "name1",
		},
		{
			name: "invalid index for []any in map[string]any",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []any{
						map[string]any{
							"id":   "1",
							"name": "name1",
						},
						map[string]any{
							"id":   "2",
							"name": "name2",
						},
					},
				},
			},
			v:       "source.items[asd].name",
			wantErr: "failed to calc index for []interface {}; err: index should be integer or one of [next, rand, last], but got `asd`",
		},
		{
			name: "slice of strings",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []string{
						"1",
						"2",
					},
				},
			},
			v:    "source.items[0]",
			want: "1",
		},
		{
			name: "slice of strings / invalid index",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []string{
						"1",
						"2",
					},
				},
			},
			v:       "source.items[asd]",
			wantErr: "failed to calc index for []string; err: index should be integer or one of [next, rand, last], but got `asd`",
		},
		{
			name: "not last segment items in path for []string",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []string{
						"1",
						"2",
					},
				},
			},
			v:       "source.items[0].key",
			wantErr: "not last segment items in path source.items[0].key",
		},
		{
			name: "extract value from array / by index / success",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []any{11, 22, 33},
				},
			},
			v:    "source.items[0]",
			want: 11,
		},
		{
			name: "slice of ints",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []int{11, 22, 33},
				},
			},
			v:    "source.items[0]",
			want: 11,
		},
		{
			name: "slice of ints",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []int64{11, 22, 33},
				},
			},
			v:    "source.items[0]",
			want: int64(11),
		},
		{
			name: "slice of ints",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []float64{11, 22, 33},
				},
			},
			v:    "source.items[0]",
			want: float64(11),
		},
		{
			name: "invalid key / in array / element / return error",
			reqMap: map[string]any{
				"source": map[string]any{
					"items": []any{11, 22, 33},
				},
			},
			v:       "source.items[0].id",
			want:    nil,
			wantErr: "not last segment items in path source.items[0].id",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			iter := NewNextIterator(0)
			got, err := GetMapValue(tt.reqMap, tt.v, iter)
			if tt.wantErr != "" {
				require.Contains(t, err.Error(), tt.wantErr)
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

var tmpJSON = `{
    "name": "John Doe",
    "age": 30,
    "isStudent": false,
    "address": {
        "street": "Main St",
        "city": "New York",
        "state": "NY",
        "postalCode": "10001",
        "geolocation": {
            "lat": 40.7128,
            "lng": 74.0060
        }
    },
    "rates": [1, 2, 3],
    "rates2": [1, "c", 3],
    "emails": [
        "johndoe@gmail.com",
        "johndoe@yahoo.com"
    ],
    "courses": [
        {
            "courseName": "Math",
            "grade": "A+"
        },
        {
            "courseName": "History",
            "grade": "B"
        }
    ],
    "skills": [
        "Programming",
        "Design",
        "Management"
    ],
    "projects": [
        {
            "title": "Restaurant Web App",
            "description": "A full stack web application for managing restaurant reservations.",
            "technologies": [
                "HTML",
                "CSS",
                "JavaScript",
                "React",
                "Node.js"
            ]
        },
        {
            "title": "E-commerce Website",
            "description": "An e-commerce platform for a small business.",
            "technologies": [
                "PHP",
                "MySQL",
                "CSS",
                "Bootstrap"
            ]
        }
    ]
}`

var tmpJSONKeys = []string{
	"name",
	"age",
	"isStudent",
	"address.street",
	"address.city",
	"address.state",
	"address.postalCode",
	"address.geolocation.lat",
	"address.geolocation.lng",
	"rates[0]",
	"rates[1]",
	"rates[2]",
	"rates2[0]",
	"rates2[1]",
	"rates2[2]",
	"emails[0]",
	"emails[1]",
	"courses[0].courseName",
	"courses[0].grade",
	"courses[1].courseName",
	"courses[1].grade",
	"skills[0]",
	"skills[1]",
	"skills[2]",
	"projects[0].title",
	"projects[0].description",
	"projects[0].technologies[next]",
	"projects[0].technologies[next]",
	"projects[0].technologies[next]",
	"projects[1].title",
	"projects[1].description",
	"projects[1].technologies[next]",
	"projects[1].technologies[next]",
	"projects[1].technologies[next]",
}

func BenchmarkGetMapValue(b *testing.B) {
	var data map[string]any
	err := json.Unmarshal([]byte(tmpJSON), &data)
	require.NoError(b, err)
	iterator := NewNextIterator(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, k := range tmpJSONKeys {
			_, err := GetMapValue(data, k, iterator)
			require.NoError(b, err)
		}
	}
}
