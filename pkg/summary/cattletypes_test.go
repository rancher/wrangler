package summary

import (
	"testing"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/stretchr/testify/assert"
)

func TestCheckRelease(t *testing.T) {
	type testCase struct {
		name     string
		input    data.Object
		expected Summary
	}

	cases := []testCase{
		{
			name: "namespaced resource for deployed app.catalog.cattle.io",
			input: data.Object{
				"apiVersion": "catalog.cattle.io/v1",
				"kind":       "App",

				"spec": map[string]any{
					"resources": []any{
						map[string]any{
							"apiVersion": "rbac.authorization.k8s.io/v1",
							"kind":       "Role",
							"name":       "monitoring-dashboard-admin",
							"namespace":  "cattle-dashboards",
						},
					},
				},

				"status": map[string]any{
					"summary": map[string]any{
						"state": "deployed",
					},
				},
			},
			expected: Summary{
				Relationships: []Relationship{
					{
						Name:       "monitoring-dashboard-admin",
						Namespace:  "cattle-dashboards",
						Kind:       "Role",
						APIVersion: "rbac.authorization.k8s.io/v1",
						Type:       "helmresource",
					},
				},
			},
		},

		{
			name: "non-namespaced resource for deployed app.catalog.cattle.io",
			input: data.Object{
				"apiVersion": "catalog.cattle.io/v1",
				"kind":       "App",

				"spec": map[string]any{
					"resources": []any{
						map[string]any{
							"apiVersion": "rbac.authorization.k8s.io/v1",
							"kind":       "ClusterRole",
							"name":       "monitoring-admin",
						},
					},
				},

				"status": map[string]any{
					"summary": map[string]any{
						"state": "deployed",
					},
				},
			},
			expected: Summary{
				Relationships: []Relationship{
					{
						Name:       "monitoring-admin",
						Kind:       "ClusterRole",
						APIVersion: "rbac.authorization.k8s.io/v1",
						Type:       "helmresource",
					},
				},
			},
		},

		{
			name: "resource for undeployed app.catalog.cattle.io",
			input: data.Object{
				"apiVersion": "catalog.cattle.io/v1",
				"kind":       "App",

				"spec": map[string]any{
					"resources": []any{
						map[string]any{
							"apiVersion": "rbac.authorization.k8s.io/v1",
							"kind":       "Role",
							"name":       "monitoring-dashboard-admin",
							"namespace":  "cattle-dashboards",
						},
					},
				},

				"status": map[string]any{
					"summary": map[string]any{
						"state": "failed",
					},
				},
			},
			expected: Summary{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := checkRelease(c.input, nil, Summary{})
			assert.Equal(t, c.expected, actual)
		})
	}
}
