package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetValueFromAny(t *testing.T) {
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
			name: "get index of top level slice",
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
			name: "slice of maps",
			data: []interface{}{
				map[string]interface{}{
					"notthisone": "val",
				},
				map[string]interface{}{
					"parent": map[string]interface{}{
						"children": []interface{}{
							"alice",
							"bob",
							"eve",
						},
					},
				},
			},
			keys:        []string{"1", "parent", "children", "0"},
			wantValue:   "alice",
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
		{
			name: "index not parseable to int",
			data: map[string]interface{}{
				"parent": map[string]interface{}{
					"children": []interface{}{
						"alice",
						"bob",
						"eve",
					},
				},
			},
			keys:        []string{"parent", "children", "notanint"},
			wantValue:   nil,
			wantSuccess: false,
		},
		{
			name: "slice blank index",
			data: []interface{}{
				"bob",
			},
			keys:        []string{""},
			wantValue:   nil,
			wantSuccess: false,
		},
		{
			name: "slice no index",
			data: []interface{}{
				"bob",
			},
			wantValue:   nil,
			wantSuccess: false,
		},
		{
			name: "keys nested too far",
			data: []interface{}{
				"alice",
				"bob",
				"eve",
			},
			keys:        []string{"2", "1"},
			wantValue:   nil,
			wantSuccess: false,
		},
		{
			name: "map blank key with value",
			data: map[string]interface{}{
				"": "bob",
			},
			keys:        []string{""},
			wantValue:   "bob",
			wantSuccess: true,
		},
		{
			name: "map blank key no value",
			data: map[string]interface{}{
				"alice": "bob",
			},
			keys:        []string{""},
			wantValue:   nil,
			wantSuccess: false,
		},
		{
			name: "map no key",
			data: map[string]interface{}{
				"": "bob",
			},
			wantValue:   nil,
			wantSuccess: false,
		},
		{
			name: "contains an array of strings at top-level",
			data: map[string]interface{}{
				"kind": "apple",
				"metadata": map[string]interface{}{
					"name": "granny-smith",
					"fields": []string{
						"a3",
						"position2",
						"more...",
					},
				},
				"data": map[string]interface{}{
					"color": "green",
				},
			},
			keys:        []string{"metadata", "fields", "1"},
			wantValue:   "position2",
			wantSuccess: true,
		},
		{
			name: "contains an array of strings at top-level",
			data: []string{
				"a4",
				"position4",
				"more...",
			},
			keys:        []string{"2"},
			wantValue:   "more...",
			wantSuccess: true,
		},
		{
			name: "index out of bounds for top-level array",
			data: []string{
				"a4",
				"position4",
				"more...",
			},
			keys:        []string{"-5"},
			wantValue:   nil,
			wantSuccess: false,
		},
		{
			name:        "doesn't handle array of ints",
			data:        []int{1, 3, 5},
			keys:        []string{"1"},
			wantValue:   nil,
			wantSuccess: false,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			gotValue, gotSuccess := GetValueFromAny(test.data, test.keys...)
			assert.Equal(t, test.wantValue, gotValue)
			assert.Equal(t, test.wantSuccess, gotSuccess)
		})
	}
}
