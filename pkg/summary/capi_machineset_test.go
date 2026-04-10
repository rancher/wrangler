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

// makeMachineDeploymentObj builds a minimal CAPI MachineDeployment data.Object
// with the given replica fields. Use -1 to omit a field.
func makeMachineDeploymentObj(specReplicas, statusReplicas, readyReplicas, upToDateReplicas int64) data.Object {
	obj := data.Object{
		"apiVersion": "cluster.x-k8s.io/v1beta2",
		"kind":       "MachineDeployment",
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
	if upToDateReplicas >= 0 {
		obj["status"].(map[string]interface{})["upToDateReplicas"] = upToDateReplicas
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

		// --- Priority 3: RollingOut ---
		// RollingOut only applies to MachineDeployment, not MachineSet.
		{
			name: "RollingOut=True on MachineDeployment takes priority over ScalingDown",
			obj:  makeMachineDeploymentObj(3, 4, 3, 2),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("RollingOut", "True", "RollingOut", "Rolling out 2 not up-to-date replicas\n* DigitaloceanMachine is not up-to-date"),
				NewCondition("ScalingDown", "True", "ScalingDown", "Scaling down from 4 to 3 replicas"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:    "rollingout",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"rolling out 2 not up-to-date replicas"},
		},
		{
			name: "RollingOut=True on MachineDeployment takes priority over ScalingUp by replicas",
			obj:  makeMachineDeploymentObj(3, 3, 1, 1),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("RollingOut", "True", "RollingOut", "Rolling out 2 not up-to-date replicas"),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:    "rollingout",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"rolling out 2 not up-to-date replicas"},
		},

		// --- Priority 4: ScalingDown ---
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

		// --- Priority 5: ScalingUp ---
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
			name: "Spec replicas exceed ready replicas — reports scalingup even when all machines exist",
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
		{
			name: "MachineDeployment with ready replicas below spec — reports scalingup before other conditions",
			obj:  makeMachineDeploymentObj(3, 3, 2, 3),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("RollingOut", "False", "NotRollingOut", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("Available", "False", "NotAvailable", "2 available replicas, at least 3 required"),
				NewCondition("MachinesReady", "Unknown", "ReadyUnknown", "* Machine test-m:\n  * NodeHealthy: Kubelet stopped posting node status"),
			},
			expectedState:    "scalingup",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling up from 2 to 3 replicas, waiting for machines to be ready"},
		},
		{
			name: "Deleting=True takes priority over RollingOut=True",
			obj:  makeMachineDeploymentObj(3, 4, 3, 2),
			conditions: []Condition{
				NewCondition("Deleting", "True", "Deleting", "being deleted"),
				NewCondition("RollingOut", "True", "RollingOut", "Rolling out 2 not up-to-date replicas"),
				NewCondition("ScalingDown", "True", "ScalingDown", "Scaling down from 4 to 3 replicas"),
			},
			expectedState:    "deleting",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"being deleted"},
		},
		{
			name: "Paused=True takes priority over RollingOut=True",
			obj:  makeMachineDeploymentObj(3, 4, 3, 2),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "True", "Paused", ""),
				NewCondition("RollingOut", "True", "RollingOut", "Rolling out 2 not up-to-date replicas"),
				NewCondition("ScalingDown", "True", "ScalingDown", "Scaling down from 4 to 3 replicas"),
			},
			expectedState:   "paused",
			expectedTransit: true,
			expectedError:   false,
		},
		{
			name: "RollingOut=False does not trigger rollingout — falls through to ScalingDown",
			obj:  makeMachineDeploymentObj(1, 2, 2, -1),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("RollingOut", "False", "NotRollingOut", ""),
				NewCondition("ScalingDown", "True", "ScalingDown", "Scaling down from 2 to 1 replicas"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:    "scalingdown",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling down from 2 to 1 replicas, waiting for machines to be deleted"},
		},
		{
			name: "RollingOut=True on MachineDeployment with missing upToDateReplicas — no message",
			obj:  makeMachineDeploymentObj(3, 4, 3, -1),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("RollingOut", "True", "RollingOut", "Rolling out 2 not up-to-date replicas"),
				NewCondition("ScalingDown", "True", "ScalingDown", "Scaling down from 4 to 3 replicas"),
			},
			expectedState:    "rollingout",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: nil,
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

// TestCheckTransitioning_CAPIMachineDeploymentDispatch verifies that
// checkTransitioning dispatches CAPI MachineDeployment objects to
// checkCAPIMachineSetAndDeploymentTransitioning and that the replica-field
// paths behave correctly for deployments (not just MachineSets).
func TestCheckTransitioning_CAPIMachineDeploymentDispatch(t *testing.T) {
	// A MachineDeployment scaling up: spec.replicas=3, status.replicas=1.
	// Even with ScalingUp=False (stale condition), the replica mismatch
	// (spec > status) should be detected via the CAPI handler.
	scalingObj := data.Object{
		"apiVersion": "cluster.x-k8s.io/v1beta2",
		"kind":       "MachineDeployment",
		"spec": map[string]interface{}{
			"replicas": int64(3),
		},
		"status": map[string]interface{}{
			"replicas":      int64(1),
			"readyReplicas": int64(1),
		},
	}
	conditions := []Condition{
		NewCondition("Deleting", "False", "NotDeleting", ""),
		NewCondition("Paused", "False", "NotPaused", ""),
		NewCondition("ScalingDown", "False", "NotScalingDown", ""),
		NewCondition("ScalingUp", "False", "NotScalingUp", ""),
	}
	result := checkTransitioning(scalingObj, conditions, Summary{})
	assert.Equal(t, "scalingup", result.State)
	assert.True(t, result.Transitioning)
	assert.Equal(t, []string{"Scaling up from 1 to 3 replicas, waiting for machines to be ready"}, result.Message)

	// A MachineDeployment in steady state: all replicas match.
	steadyObj := data.Object{
		"apiVersion": "cluster.x-k8s.io/v1beta2",
		"kind":       "MachineDeployment",
		"spec": map[string]interface{}{
			"replicas": int64(2),
		},
		"status": map[string]interface{}{
			"replicas":      int64(2),
			"readyReplicas": int64(2),
		},
	}
	steadyConditions := []Condition{
		NewCondition("Deleting", "False", "NotDeleting", ""),
		NewCondition("Paused", "False", "NotPaused", ""),
		NewCondition("ScalingDown", "False", "NotScalingDown", ""),
		NewCondition("ScalingUp", "False", "NotScalingUp", ""),
		NewCondition("MachinesReady", "True", "Ready", ""),
	}
	steadyResult := checkTransitioning(steadyObj, steadyConditions, Summary{})
	assert.Empty(t, steadyResult.State, "steady-state MachineDeployment should pass through")
	assert.False(t, steadyResult.Transitioning)
	assert.False(t, steadyResult.Error)

	// A MachineDeployment doing a rolling upgrade: RollingOut=True should
	// take priority over the ScalingDown condition and replica mismatch.
	rollingObj := data.Object{
		"apiVersion": "cluster.x-k8s.io/v1beta2",
		"kind":       "MachineDeployment",
		"spec": map[string]interface{}{
			"replicas": int64(3),
		},
		"status": map[string]interface{}{
			"replicas":         int64(4),
			"readyReplicas":    int64(3),
			"upToDateReplicas": int64(2),
		},
	}
	rollingConditions := []Condition{
		NewCondition("Deleting", "False", "NotDeleting", ""),
		NewCondition("Paused", "False", "NotPaused", ""),
		NewCondition("RollingOut", "True", "RollingOut", "Rolling out 2 not up-to-date replicas"),
		NewCondition("ScalingDown", "True", "ScalingDown", "Scaling down from 4 to 3 replicas"),
		NewCondition("ScalingUp", "False", "NotScalingUp", ""),
	}
	rollingResult := checkTransitioning(rollingObj, rollingConditions, Summary{})
	assert.Equal(t, "rollingout", rollingResult.State, "rolling upgrade should produce state=rollingout")
	assert.True(t, rollingResult.Transitioning)
	assert.Equal(t, []string{"rolling out 2 not up-to-date replicas"}, rollingResult.Message)
}
