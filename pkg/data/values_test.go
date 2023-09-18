package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		data        interface{}
		keys        []string
		wantValue   interface{}
		wantSuccess bool
	}{
		{
			name:        "nil map",
			data:        nil,
			keys:        []string{"somekey"},
			wantValue:   nil,
			wantSuccess: false,
		},
		{
			name: "key is not in map",
			data: map[string]interface{}{
				"realKey": "realVal",
			},
			keys:        []string{"badKey"},
			wantValue:   nil,
			wantSuccess: false,
		},
		{
			name: "key is in first level of map",
			data: map[string]interface{}{
				"realKey": "realVal",
			},
			keys:        []string{"realKey"},
			wantValue:   "realVal",
			wantSuccess: true,
		},
		{
			name: "key is nested in map",
			data: map[string]interface{}{
				"parent": map[string]interface{}{
					"child": map[string]interface{}{
						"grandchild": "someValue",
					},
				},
			},
			keys:        []string{"parent", "child", "grandchild"},
			wantValue:   "someValue",
			wantSuccess: true,
		},
		{
			name: "incorrected nested key",
			data: map[string]interface{}{
				"parent": map[string]interface{}{
					"child": map[string]interface{}{
						"grandchild": "someValue",
					},
				},
			},
			keys:        []string{"parent", "grandchild", "child"},
			wantValue:   nil,
			wantSuccess: false,
		},
		{
			name: "get index of slice",
			data: map[string]interface{}{
				"parent": map[string]interface{}{
					"children": []interface{}{
						"alice",
						"bob",
						"eve",
					},
				},
			},
			keys:        []string{"parent", "children", "2"},
			wantValue:   "eve",
			wantSuccess: true,
		},
		{
			name: "get index of top levelslice",
			data: []interface{}{
				"alice",
				"bob",
				"eve",
			},
			keys:        []string{"2"},
			wantValue:   "eve",
			wantSuccess: true,
		},
		{
			name: "index is too big",
			data: map[string]interface{}{
				"parent": map[string]interface{}{
					"children": []interface{}{
						"alice",
						"bob",
						"eve",
					},
				},
			},
			keys:        []string{"parent", "children", "3"},
			wantValue:   nil,
			wantSuccess: false,
		},
		{
			name: "index is negative",
			data: map[string]interface{}{
				"parent": map[string]interface{}{
					"children": []interface{}{
						"alice",
						"bob",
						"eve",
					},
				},
			},
			keys:        []string{"parent", "children", "-3"},
			wantValue:   nil,
			wantSuccess: false,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			gotValue, gotSuccess := GetValue(test.data, test.keys...)
			assert.Equal(t, test.wantValue, gotValue)
			assert.Equal(t, test.wantSuccess, gotSuccess)
		})
	}
}
