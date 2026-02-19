package summary

import (
	"testing"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
			name: "annotation conditions with CAPI v1beta2 standard conditions",
			input: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
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
			expected: []data.Object{},
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

func TestNormalizeConditions(t *testing.T) {
	tests := []struct {
		name                     string
		input                    *unstructured.Unstructured
		expectedStatusConditions []interface{}
	}{
		{
			name: "normalizes standard status conditions",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":               "Available",
								"status":             "True",
								"lastTransitionTime": "2024-01-01T00:00:00Z",
							},
						},
					},
				},
			},
			expectedStatusConditions: []interface{}{
				map[string]interface{}{
					"type":               "Available",
					"status":             "True",
					"lastTransitionTime": "2024-01-01T00:00:00Z",
					"lastUpdateTime":     "2024-01-01T00:00:00Z",
					"error":              false,
					"transitioning":      false,
				},
			},
		},
		{
			name: "normalizes CAPI v1beta2 standard conditions",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "cluster.x-k8s.io/v1beta2",
					"kind":       "Cluster",
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":               "Available",
								"status":             "True",
								"lastTransitionTime": "2024-01-02T00:00:00Z",
							},
						},
					},
				},
			},
			expectedStatusConditions: []interface{}{
				map[string]interface{}{
					"type":               "Available",
					"status":             "True",
					"lastTransitionTime": "2024-01-02T00:00:00Z",
					"lastUpdateTime":     "2024-01-02T00:00:00Z",
					"error":              false,
					"transitioning":      false,
				},
			},
		},
		{
			name: "preserves lastUpdateTime if already set",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":               "Available",
								"status":             "True",
								"lastTransitionTime": "2024-01-01T00:00:00Z",
								"lastUpdateTime":     "2024-01-02T00:00:00Z",
							},
						},
					},
				},
			},
			expectedStatusConditions: []interface{}{
				map[string]interface{}{
					"type":               "Available",
					"status":             "True",
					"lastTransitionTime": "2024-01-01T00:00:00Z",
					"lastUpdateTime":     "2024-01-02T00:00:00Z",
					"error":              false,
					"transitioning":      false,
				},
			},
		},
		{
			name: "handles empty status",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
				},
			},
			expectedStatusConditions: nil,
		},
		{
			name: "sets transitioning for False status condition",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":               "Available",
								"status":             "False",
								"reason":             "MinimumReplicasUnavailable",
								"message":            "Deployment does not have minimum availability",
								"lastTransitionTime": "2024-01-01T00:00:00Z",
							},
						},
					},
				},
			},
			expectedStatusConditions: []interface{}{
				map[string]interface{}{
					"type":               "Available",
					"status":             "False",
					"reason":             "MinimumReplicasUnavailable",
					"message":            "Deployment does not have minimum availability",
					"lastTransitionTime": "2024-01-01T00:00:00Z",
					"lastUpdateTime":     "2024-01-01T00:00:00Z",
					"error":              false,
					"transitioning":      true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NormalizeConditions(tt.input)

			// Check standard status conditions
			statusConditions, _, _ := unstructured.NestedSlice(tt.input.Object, "status", "conditions")
			if tt.expectedStatusConditions == nil {
				assert.Nil(t, statusConditions)
			} else {
				assert.Equal(t, tt.expectedStatusConditions, statusConditions)
			}
		})
	}
}

func TestNormalizeConditionsNonUnstructured(t *testing.T) {
	// Test that NormalizeConditions handles non-unstructured objects gracefully
	// Passing nil should not panic because the type assertion will fail
	NormalizeConditions(nil) // Should not panic
}
