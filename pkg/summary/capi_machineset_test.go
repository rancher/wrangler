package summary

import (
	"testing"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/stretchr/testify/assert"
)

func TestIsCAPIMachineSet(t *testing.T) {
	tests := []struct {
		name     string
		obj      data.Object
		expected bool
	}{
		{
			name: "CAPI v1beta2 MachineSet",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "MachineSet",
			},
			expected: true,
		},
		{
			name: "CAPI v1beta1 MachineSet",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta1",
				"kind":       "MachineSet",
			},
			expected: true,
		},
		{
			name: "CAPI v1beta2 Machine (not MachineSet)",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "Machine",
			},
			expected: false,
		},
		{
			name: "CAPI v1beta2 Cluster",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "Cluster",
			},
			expected: false,
		},
		{
			name: "non-CAPI MachineSet kind",
			obj: data.Object{
				"apiVersion": "apps/v1",
				"kind":       "ReplicaSet",
			},
			expected: false,
		},
		{
			name:     "empty object",
			obj:      data.Object{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isCAPIMachineSet(tt.obj))
		})
	}
}

// makeMachineSetObj builds a minimal CAPI MachineSet data.Object with the
// given replica fields. Use -1 to omit a field (simulating it not being set).
func makeMachineSetObj(specReplicas, statusReplicas, readyReplicas int64) data.Object {
	obj := data.Object{
		"apiVersion": "cluster.x-k8s.io/v1beta2",
		"kind":       "MachineSet",
		"spec":       map[string]interface{}{},
		"status":     map[string]interface{}{},
	}
	if specReplicas >= 0 {
		obj["spec"].(map[string]interface{})["replicas"] = specReplicas
	}
	if statusReplicas >= 0 {
		obj["status"].(map[string]interface{})["replicas"] = statusReplicas
	}
	if readyReplicas >= 0 {
		obj["status"].(map[string]interface{})["readyReplicas"] = readyReplicas
	}
	return obj
}

func TestCheckCAPIMachineSetAndDeploymentTransitioning(t *testing.T) {
	tests := []struct {
		name             string
		obj              data.Object
		conditions       []Condition
		expectedState    string
		expectedTransit  bool
		expectedError    bool
		expectedMessages []string
	}{
		// --- Priority 1: Deleting ---
		{
			name: "Deleting=True takes absolute priority",
			obj:  makeMachineSetObj(2, 2, 2),
			conditions: []Condition{
				NewCondition("Deleting", "True", "Deleting", "machineset is being deleted"),
				NewCondition("ScalingUp", "True", "ScalingUp", "Scaling up from 1 to 2 replicas"),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:    "deleting",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"machineset is being deleted"},
		},
		{
			name: "Deleting=True with empty message",
			obj:  makeMachineSetObj(2, 2, 2),
			conditions: []Condition{
				NewCondition("Deleting", "True", "Deleting", ""),
			},
			expectedState:    "deleting",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: nil,
		},
		{
			name: "Deleting=False is ignored",
			obj:  makeMachineSetObj(2, 2, 2),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("MachinesReady", "True", "Ready", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},

		// --- Priority 2: Paused ---
		{
			name: "Paused=True takes priority over scaling conditions",
			obj:  makeMachineSetObj(2, 1, 1),
			conditions: []Condition{
				NewCondition("Paused", "True", "Paused", "machineset is paused"),
				NewCondition("ScalingUp", "True", "ScalingUp", "Scaling up from 1 to 2 replicas"),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:   "paused",
			expectedTransit: true,
			expectedError:   false,
		},
		{
			name: "Paused=False is ignored",
			obj:  makeMachineSetObj(2, 2, 2),
			conditions: []Condition{
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("MachinesReady", "True", "Ready", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},

		// --- Priority 3: ScalingDown ---
		{
			name: "ScalingDown=True — message always constructed from replicas",
			obj:  makeMachineSetObj(1, 2, 2),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "True", "ScalingDown", "Scaling down from 2 to 1 replicas"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:    "scalingdown",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling down from 2 to 1 replicas, waiting for machines to be deleted"},
		},
		{
			name: "ScalingDown=True with empty condition message — constructs from replicas",
			obj:  makeMachineSetObj(1, 2, 2),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "True", "ScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:    "scalingdown",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling down from 2 to 1 replicas, waiting for machines to be deleted"},
		},
		{
			name: "ScalingDown detected by replica mismatch only (stale condition)",
			obj:  makeMachineSetObj(1, 2, 2),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:    "scalingdown",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling down from 2 to 1 replicas, waiting for machines to be deleted"},
		},
		{
			name: "ScalingDown detected by replica mismatch",
			obj:  makeMachineSetObj(1, 2, 1),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("MachinesReady", "True", "Ready", ""),
			},
			expectedState:    "scalingdown",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling down from 2 to 1 replicas, waiting for machines to be deleted"},
		},
		{
			name: "ScalingDown takes priority over ScalingUp when both detectable",
			obj:  makeMachineSetObj(1, 3, 0),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "True", "ScalingDown", "Scaling down from 3 to 1 replicas"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:    "scalingdown",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling down from 3 to 1 replicas, waiting for machines to be deleted"},
		},

		// --- Priority 4: ScalingUp ---
		{
			name: "ScalingUp=True — message always constructed from replicas",
			obj:  makeMachineSetObj(2, 1, 1),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "True", "ScalingUp", "Scaling up from 1 to 2 replicas"),
			},
			expectedState:    "scalingup",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling up from 1 to 2 replicas, waiting for machines to be ready"},
		},
		{
			name: "ScalingUp=True with empty condition message — constructs from replicas",
			obj:  makeMachineSetObj(2, 1, 1),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "True", "ScalingUp", ""),
			},
			expectedState:    "scalingup",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling up from 1 to 2 replicas, waiting for machines to be ready"},
		},
		{
			name: "ScalingUp detected by readyReplicas mismatch only",
			obj:  makeMachineSetObj(2, 1, 1),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:    "scalingup",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling up from 1 to 2 replicas, waiting for machines to be ready"},
		},
		{
			name: "ScalingUp detected by readyReplicas mismatch",
			obj:  makeMachineSetObj(2, 2, 1),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("MachinesReady", "True", "Ready", ""),
			},
			expectedState:    "scalingup",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling up from 1 to 2 replicas, waiting for machines to be ready"},
		},

		// --- Pass through (steady state) ---
		{
			name: "Steady state — all replicas match → pass through",
			obj:  makeMachineSetObj(2, 2, 2),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("MachinesReady", "True", "Ready", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},

		// --- Edge cases ---
		{
			name:            "No conditions → pass through",
			obj:             makeMachineSetObj(1, 1, 1),
			conditions:      []Condition{},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},
		{
			name: "Missing replica fields → pass through (no panic)",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "MachineSet",
			},
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},
		{
			name: "ScalingUp=True but no replica fields → scalingup, no message",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "MachineSet",
			},
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "True", "ScalingUp", ""),
			},
			expectedState:    "scalingup",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: nil,
		},
		{
			name: "ScalingUp from zero readyReplicas",
			obj:  makeMachineSetObj(2, 0, 0),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "True", "ScalingUp", ""),
			},
			expectedState:    "scalingup",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling up from 0 to 2 replicas, waiting for machines to be ready"},
		},
		{
			name: "ScalingUp with larger replica counts (3 → 5)",
			obj:  makeMachineSetObj(5, 3, 3),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "True", "ScalingUp", ""),
			},
			expectedState:    "scalingup",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling up from 3 to 5 replicas, waiting for machines to be ready"},
		},
		{
			name: "Deleting=True takes priority over active scale-down",
			obj:  makeMachineSetObj(1, 2, 2),
			conditions: []Condition{
				NewCondition("Deleting", "True", "Deleting", "machineset cleanup in progress"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "True", "ScalingDown", "Scaling down from 2 to 1 replicas"),
			},
			expectedState:    "deleting",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"machineset cleanup in progress"},
		},
		{
			name: "Paused=True takes priority over scale-up detected by replicas",
			obj:  makeMachineSetObj(3, 1, 1),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "True", "Paused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:   "paused",
			expectedTransit: true,
			expectedError:   false,
		},
		{
			name: "ScalingDown=True but no replica fields → scalingdown, no message",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "MachineSet",
			},
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "True", "ScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:    "scalingdown",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkCAPIMachineSetAndDeploymentTransitioning(tt.obj, tt.conditions, Summary{})
			assert.Equal(t, tt.expectedState, result.State, "state mismatch")
			assert.Equal(t, tt.expectedTransit, result.Transitioning, "transitioning mismatch")
			assert.Equal(t, tt.expectedError, result.Error, "error mismatch")
			if tt.expectedMessages != nil {
				assert.Equal(t, tt.expectedMessages, result.Message, "messages mismatch")
			} else {
				assert.Empty(t, result.Message, "expected no messages")
			}
		})
	}
}

func TestIsCAPIMachineDeployment(t *testing.T) {
	tests := []struct {
		name     string
		obj      data.Object
		expected bool
	}{
		{
			name: "CAPI v1beta2 MachineDeployment",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "MachineDeployment",
			},
			expected: true,
		},
		{
			name: "CAPI v1beta1 MachineDeployment",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta1",
				"kind":       "MachineDeployment",
			},
			expected: true,
		},
		{
			name: "CAPI v1beta2 MachineSet (not MachineDeployment)",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "MachineSet",
			},
			expected: false,
		},
		{
			name: "CAPI v1beta2 Cluster",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "Cluster",
			},
			expected: false,
		},
		{
			name: "non-CAPI Deployment kind",
			obj: data.Object{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
			},
			expected: false,
		},
		{
			name:     "empty object",
			obj:      data.Object{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isCAPIMachineDeployment(tt.obj))
		})
	}
}
