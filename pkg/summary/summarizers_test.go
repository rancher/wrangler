package summary

import (
	"os"
	"testing"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/stretchr/testify/assert"
)

func TestCheckGVKErrors(t *testing.T) {
	type input struct {
		data       data.Object
		conditions []Condition
		summary    Summary
	}

	type output struct {
		summary Summary
		handled bool
	}

	testCases := []struct {
		name           string
		loadConditions func()
		input          input
		expected       output
	}{
		{
			name: "gvk not detected - summary remains the same",
			input: input{
				data: data.Object{},
				summary: Summary{
					State: "testing",
					Error: false,
				},
			},
			expected: output{
				summary: Summary{
					State: "testing",
					Error: false,
				},
				handled: false,
			},
		},
		{
			name: "gvk not found - summary remains the same",
			input: input{
				data: data.Object{
					"APIVersion": "sample.cattle.io/v1",
					"Kind":       "Sample",
				},
				summary: Summary{
					State: "testing",
					Error: false,
				},
			},
			expected: output{
				summary: Summary{
					State: "testing",
					Error: false,
				},
				handled: false,
			},
		},
		{
			name: "gvk found, no conditions provided",
			input: input{
				data: data.Object{
					"APIVersion": "helm.cattle.io/v1",
					"Kind":       "HelmChart",
				},
				summary: Summary{
					State: "testing",
					Error: false,
				},
			},
			expected: output{
				summary: Summary{
					State: "testing",
					Error: false,
				},
				handled: false,
			},
		},
		{
			name: "gvk found, condition not found",
			input: input{
				data: data.Object{
					"APIVersion": "helm.cattle.io/v1",
					"Kind":       "HelmChart",
				},
				conditions: []Condition{
					NewCondition("JobFailed", "True", "", ""),
				},
				summary: Summary{
					State: "testing",
					Error: false,
				},
			},
			expected: output{
				summary: Summary{
					State: "testing",
					Error: false,
				},
				handled: false,
			},
		},
		{
			name: "gvk found, condition is error",
			input: input{
				data: data.Object{
					"APIVersion": "helm.cattle.io/v1",
					"Kind":       "HelmChart",
				},
				conditions: []Condition{
					NewCondition("Failed", "True", "", ""),
				},
				summary: Summary{
					State: "testing",
					Error: false,
				},
			},
			expected: output{
				summary: Summary{
					State: "testing",
					Error: true,
				},
				handled: true,
			},
		},
		{
			name: "gvk found, condition is not an error",
			input: input{
				data: data.Object{
					"APIVersion": "helm.cattle.io/v1",
					"Kind":       "HelmChart",
				},
				conditions: []Condition{
					NewCondition("Failed", "False", "", ""),
				},
				summary: Summary{
					State: "testing",
					Error: false,
				},
			},
			expected: output{
				summary: Summary{
					State: "testing",
					Error: false,
				},
				handled: true,
			},
		},
		{
			name: "load conditions - gvk not found",
			input: input{
				data: data.Object{
					"APIVersion": "helm.cattle.io/v1",
					"Kind":       "HelmChart",
				},
				conditions: []Condition{
					NewCondition("Failed", "False", "", ""),
				},
				summary: Summary{
					State: "testing",
					Error: false,
				},
			},
			expected: output{
				summary: Summary{
					State: "testing",
					Error: false,
				},
				handled: false,
			},
			loadConditions: func() {
				os.Setenv("CATTLE_WRANGLER_CHECK_GVK_ERROR_MAPPING", `
					[
						{
							"gvk": "sample.cattle.io/v1/Sample",
							"conditionMapping": [
								{
									"type": "Failed",
									"status": "True",
									"error": true
								}
							]
						}
					]
				`)
			},
		},
		{
			name: "load conditions - gvk found - is not an error",
			input: input{
				data: data.Object{
					"APIVersion": "sample.cattle.io/v1",
					"Kind":       "Sample",
				},
				conditions: []Condition{
					NewCondition("Failed", "False", "", ""),
				},
				summary: Summary{
					State: "testing",
					Error: false,
				},
			},
			expected: output{
				summary: Summary{
					State: "testing",
					Error: false,
				},
				handled: true,
			},
			loadConditions: func() {
				os.Setenv("CATTLE_WRANGLER_CHECK_GVK_ERROR_MAPPING", `
					[
						{
							"gvk": "sample.cattle.io/v1/Sample",
							"conditionMapping": [
								{
									"type": "Failed",
									"status": "False",
									"error": false
								}
							]
						}
					]
				`)
			},
		},
		{
			name: "load conditions - gvk found - is error",
			input: input{
				data: data.Object{
					"APIVersion": "sample.cattle.io/v1",
					"Kind":       "Sample",
				},
				conditions: []Condition{
					NewCondition("Failed", "True", "", ""),
				},
				summary: Summary{
					State: "testing",
					Error: false,
				},
			},
			expected: output{
				summary: Summary{
					State: "testing",
					Error: true,
				},
				handled: true,
			},
			loadConditions: func() {
				os.Setenv("CATTLE_WRANGLER_CHECK_GVK_ERROR_MAPPING", `
					[
						{
							"gvk": "sample.cattle.io/v1/Sample",
							"conditionMapping": [
								{
									"type": "Failed",
									"status": "True",
									"error": true
								}
							]
						}
					]
				`)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.loadConditions != nil {
				tc.loadConditions()
			}
			initializeCheckGVKError()
			summary, handled := checkGVKErrors(tc.input.data, tc.input.conditions, tc.input.summary)

			assert.Equal(t, tc.expected.summary.Error, summary.Error)
			assert.Equal(t, tc.expected.handled, handled)
		})
	}

}
