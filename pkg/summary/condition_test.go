package summary

import (
	"testing"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/stretchr/testify/assert"
)

func TestGetRawConditions(t *testing.T) {
	tests := []struct {
		name     string
		input    data.Object
		expected []data.Object
	}{
		{
			name: "standard conditions in status field",
			input: data.Object{
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
						map[string]interface{}{
							"type":   "Available",
							"status": "False",
						},
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "Ready",
					"status": "True",
				},
				{
					"type":   "Available",
					"status": "False",
				},
			},
		},
		{
			name: "deprecated CAPI v1beta1 conditions take priority",
			input: data.Object{
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
					},
					"deprecated": map[string]interface{}{
						"v1beta1": map[string]interface{}{
							"conditions": []interface{}{
								map[string]interface{}{
									"type":   "LegacyReady",
									"status": "True",
								},
							},
						},
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "LegacyReady",
					"status": "True",
				},
			},
		},
		{
			name: "annotation conditions are appended to status conditions",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": `{"conditions":[{"type":"Annotation","status":"True"}]}`,
					},
				},
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "Ready",
					"status": "True",
				},
				{
					"type":   "Annotation",
					"status": "True",
				},
			},
		},
		{
			name: "annotation conditions with deprecated CAPI conditions",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": `{"conditions":[{"type":"Annotation","status":"True"}]}`,
					},
				},
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
					},
					"deprecated": map[string]interface{}{
						"v1beta1": map[string]interface{}{
							"conditions": []interface{}{
								map[string]interface{}{
									"type":   "LegacyReady",
									"status": "True",
								},
							},
						},
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "LegacyReady",
					"status": "True",
				},
				{
					"type":   "Annotation",
					"status": "True",
				},
			},
		},
		{
			name: "only annotation conditions when status field is empty",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": `{"conditions":[{"type":"Annotation","status":"True"}]}`,
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "Annotation",
					"status": "True",
				},
			},
		},
		{
			name:     "no conditions at all",
			input:    data.Object{},
			expected: nil,
		},
		{
			name: "invalid annotation JSON is ignored",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": `invalid json`,
					},
				},
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "Ready",
					"status": "True",
				},
			},
		},
		{
			name: "empty annotation string is ignored",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": "",
					},
				},
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "Ready",
					"status": "True",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRawConditions(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStatusConditions(t *testing.T) {
	tests := []struct {
		name     string
		input    data.Object
		expected []data.Object
	}{
		{
			name: "returns standard status conditions",
			input: data.Object{
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "Ready",
					"status": "True",
				},
			},
		},
		{
			name: "prioritizes deprecated v1beta1 conditions",
			input: data.Object{
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
					},
					"deprecated": map[string]interface{}{
						"v1beta1": map[string]interface{}{
							"conditions": []interface{}{
								map[string]interface{}{
									"type":   "LegacyReady",
									"status": "False",
								},
							},
						},
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "LegacyReady",
					"status": "False",
				},
			},
		},
		{
			name: "returns empty slice when no conditions exist",
			input: data.Object{
				"status": map[string]interface{}{},
			},
			expected: nil,
		},
		{
			name:     "returns empty slice when status field doesn't exist",
			input:    data.Object{},
			expected: nil,
		},
		{
			name: "returns empty slice when deprecated path exists but conditions are empty",
			input: data.Object{
				"status": map[string]interface{}{
					"deprecated": map[string]interface{}{
						"v1beta1": map[string]interface{}{
							"conditions": []interface{}{},
						},
					},
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "Ready",
					"status": "True",
				},
			},
		},
		{
			name: "handles multiple deprecated conditions",
			input: data.Object{
				"status": map[string]interface{}{
					"deprecated": map[string]interface{}{
						"v1beta1": map[string]interface{}{
							"conditions": []interface{}{
								map[string]interface{}{
									"type":   "Ready",
									"status": "True",
								},
								map[string]interface{}{
									"type":   "Available",
									"status": "False",
								},
							},
						},
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "Ready",
					"status": "True",
				},
				{
					"type":   "Available",
					"status": "False",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStatusConditions(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAnnotationConditions(t *testing.T) {
	tests := []struct {
		name     string
		input    data.Object
		expected []data.Object
	}{
		{
			name: "parses valid annotation conditions",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": `{"conditions":[{"type":"Ready","status":"True"}]}`,
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "Ready",
					"status": "True",
				},
			},
		},
		{
			name: "parses multiple annotation conditions",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": `{"conditions":[{"type":"Ready","status":"True"},{"type":"Available","status":"False"}]}`,
					},
				},
			},
			expected: []data.Object{
				{
					"type":   "Ready",
					"status": "True",
				},
				{
					"type":   "Available",
					"status": "False",
				},
			},
		},
		{
			name: "returns empty slice when annotation is empty string",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": "",
					},
				},
			},
			expected: []data.Object{},
		},
		{
			name: "returns empty slice when annotation doesn't exist",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{},
				},
			},
			expected: []data.Object{},
		},
		{
			name:     "returns empty slice when metadata doesn't exist",
			input:    data.Object{},
			expected: []data.Object{},
		},
		{
			name: "returns empty slice when JSON is invalid",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": `{invalid json}`,
					},
				},
			},
			expected: []data.Object{},
		},
		{
			name: "returns empty slice when JSON is valid but has no conditions",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": `{"other":"field"}`,
					},
				},
			},
			expected: []data.Object{},
		},
		{
			name: "returns empty slice when conditions array is empty",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": `{"conditions":[]}`,
					},
				},
			},
			expected: []data.Object{},
		},
		{
			name: "handles complex condition objects",
			input: data.Object{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"cattle.io/status": `{"conditions":[{"type":"Ready","status":"True","reason":"AllGood","message":"Everything is fine","lastTransitionTime":"2024-01-01T00:00:00Z"}]}`,
					},
				},
			},
			expected: []data.Object{
				{
					"type":               "Ready",
					"status":             "True",
					"reason":             "AllGood",
					"message":            "Everything is fine",
					"lastTransitionTime": "2024-01-01T00:00:00Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAnnotationConditions(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
