package summary

import (
	"testing"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/stretchr/testify/assert"
)

func TestIsCAPICluster(t *testing.T) {
	tests := []struct {
		name     string
		obj      data.Object
		expected bool
	}{
		{
			name: "CAPI v1beta2 Cluster",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "Cluster",
			},
			expected: true,
		},
		{
			name: "CAPI v1beta1 Cluster",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta1",
				"kind":       "Cluster",
			},
			expected: true,
		},
		{
			name: "CAPI v1beta2 Machine (not Cluster)",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "Machine",
			},
			expected: false,
		},
		{
			name: "CAPI v1beta2 MachineSet (not Cluster)",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "MachineSet",
			},
			expected: false,
		},
		{
			name: "Rancher management Cluster (not CAPI)",
			obj: data.Object{
				"apiVersion": "management.cattle.io/v3",
				"kind":       "Cluster",
			},
			expected: false,
		},
		{
			name: "non-CAPI kind",
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
			assert.Equal(t, tt.expected, isCAPICluster(tt.obj))
		})
	}
}

// makeClusterObj builds a minimal CAPI Cluster data.Object with the given
// worker replica fields under status.workers. Use -1 to omit a field.
func makeClusterObj(desiredReplicas, replicas, readyReplicas, availableReplicas, upToDateReplicas int64) data.Object {
	obj := data.Object{
		"apiVersion": "cluster.x-k8s.io/v1beta2",
		"kind":       "Cluster",
		"status": map[string]interface{}{
			"workers": map[string]interface{}{},
		},
	}
	workers := obj["status"].(map[string]interface{})["workers"].(map[string]interface{})
	if desiredReplicas >= 0 {
		workers["desiredReplicas"] = desiredReplicas
	}
	if replicas >= 0 {
		workers["replicas"] = replicas
	}
	if readyReplicas >= 0 {
		workers["readyReplicas"] = readyReplicas
	}
	if availableReplicas >= 0 {
		workers["availableReplicas"] = availableReplicas
	}
	if upToDateReplicas >= 0 {
		workers["upToDateReplicas"] = upToDateReplicas
	}
	return obj
}

func TestCheckCAPIClusterTransitioning(t *testing.T) {
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
			name: "Deleting=True takes absolute priority over everything",
			obj:  makeClusterObj(1, 1, 1, 1, -1),
			conditions: []Condition{
				NewCondition("Available", "False", "NotAvailable", "* Deleting: ..."),
				NewCondition("ScalingDown", "True", "ScalingDown", "* MachineDeployment md: Scaling down from 1 to 0 replicas"),
				NewCondition("Deleting", "True", "WaitingForWorkersDeletion", "* MachineDeployments: md\n* MachineSets: ms"),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:    "deleting",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"waiting for workers deletion"},
		},
		{
			name: "Deleting=True with empty message",
			obj:  makeClusterObj(1, 1, 1, 1, -1),
			conditions: []Condition{
				NewCondition("Deleting", "True", "Deleting", ""),
			},
			expectedState:    "deleting",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: nil,
		},
		{
			name: "Deleting=False does not trigger removing",
			obj:  makeClusterObj(2, 2, 2, 2, -1),
			conditions: []Condition{
				NewCondition("Available", "True", "Available", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},

		// --- Priority 2: Paused ---
		{
			name: "Paused=True takes priority over scaling and Available",
			obj:  makeClusterObj(3, 2, 2, 2, -1),
			conditions: []Condition{
				NewCondition("Available", "False", "NotAvailable", "something"),
				NewCondition("ScalingUp", "True", "ScalingUp", "scaling"),
				NewCondition("Paused", "True", "Paused", "cluster is paused"),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:   "paused",
			expectedTransit: true,
			expectedError:   false,
		},
		{
			name: "Paused=False is ignored",
			obj:  makeClusterObj(2, 2, 2, 2, -1),
			conditions: []Condition{
				NewCondition("Available", "True", "Available", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},

		// --- Priority 3: Rolling out ---
		{
			name: "RollingOut=True takes priority over ScalingDown during rolling upgrade",
			obj:  makeClusterObj(4, 5, 5, 5, 3),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("RollingOut", "True", "RollingOut", "* MachineDeployment do-check-jiaqi-dow:\n  * Rolling out 2 not up-to-date replicas"),
				NewCondition("ScalingDown", "True", "ScalingDown", "* MachineDeployment do-check-jiaqi-dow: Scaling down from 4 to 3 replicas"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("Available", "True", "Available", ""),
			},
			expectedState:    "rollingout",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"rolling out 2 not up-to-date replicas"},
		},
		{
			name: "RollingOut=True takes priority over ScalingUp during rolling upgrade",
			obj:  makeClusterObj(4, 4, 3, 3, 2),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("RollingOut", "True", "RollingOut", "* MachineDeployment md:\n  * Rolling out 2 not up-to-date replicas"),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("Available", "False", "NotAvailable", "something"),
			},
			expectedState:    "rollingout",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"rolling out 2 not up-to-date replicas"},
		},

		// --- Priority 4: ScalingDown ---
		{
			name: "ScalingDown=True — message constructed from worker replicas",
			obj:  makeClusterObj(2, 3, 3, 3, -1),
			conditions: []Condition{
				NewCondition("Available", "True", "Available", ""),
				NewCondition("ScalingDown", "True", "ScalingDown", "* MachineDeployment md: Scaling down from 2 to 1 replicas"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling down from 3 to 2 machines"},
		},
		{
			name: "ScalingDown detected by replica mismatch only (stale condition)",
			obj:  makeClusterObj(2, 3, 2, 2, -1),
			conditions: []Condition{
				NewCondition("Available", "True", "Available", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling down from 3 to 2 machines"},
		},
		{
			name: "ScalingDown takes priority over ScalingUp when both detectable",
			obj:  makeClusterObj(1, 3, 0, 0, -1),
			conditions: []Condition{
				NewCondition("ScalingDown", "True", "ScalingDown", "scaling down"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling down from 3 to 1 machines"},
		},

		// --- Priority 5: ScalingUp ---
		{
			name: "ScalingUp=True — scale-up scenario (2→3 workers)",
			obj:  makeClusterObj(3, 2, 2, 2, -1),
			conditions: []Condition{
				NewCondition("Available", "False", "NotAvailable", "* WorkersAvailable: insufficient replicas"),
				NewCondition("ScalingUp", "True", "ScalingUp", "* MachineDeployment md: Scaling up from 1 to 2 replicas"),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling up from 2 to 3 machines"},
		},
		{
			name: "ScalingUp detected by replica mismatch (stale condition)",
			obj:  makeClusterObj(3, 2, 2, 2, -1),
			conditions: []Condition{
				NewCondition("Available", "False", "NotAvailable", "something"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling up from 2 to 3 machines"},
		},
		{
			name: "ScalingUp during cluster creation (0→1 workers)",
			obj:  makeClusterObj(1, -1, -1, -1, -1),
			conditions: []Condition{
				NewCondition("Available", "False", "NotAvailable", "* WorkersAvailable: waiting"),
				NewCondition("ScalingUp", "True", "ScalingUp", "* MachineDeployment md: Scaling up from 0 to 1 replicas"),
				NewCondition("ControlPlaneInitialized", "False", "NotInitialized", "Control plane not yet initialized"),
				NewCondition("ControlPlaneAvailable", "False", "NotAvailable", "CP not available"),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: nil, // no readyReplicas field → no message
		},
		{
			name: "ScalingUp from zero readyReplicas during creation",
			obj:  makeClusterObj(1, 0, 0, 0, -1),
			conditions: []Condition{
				NewCondition("Available", "False", "NotAvailable", "not available"),
				NewCondition("ScalingUp", "True", "ScalingUp", "* MachineDeployment md: Scaling up from 0 to 1 replicas"),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling up from 0 to 1 machines"},
		},
		{
			name: "Desired workers exceed ready workers — reports scalingup before Available=False",
			obj:  makeClusterObj(3, 3, 2, 2, 3),
			conditions: []Condition{
				NewCondition("Available", "False", "NotAvailable", "* MachineDeployment md: 2 available, 3 required"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("RollingOut", "False", "NotRollingOut", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling up from 2 to 3 machines"},
		},

		// --- Priority 6: Available=False ---
		{
			name: "Available=False without any scaling → updating",
			obj:  makeClusterObj(2, 2, 2, 2, -1),
			conditions: []Condition{
				NewCondition("Available", "False", "NotAvailable", "* RemoteConnectionProbe: Remote connection not established yet"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"establishing connection to control plane"},
		},
		{
			name: "Available=False with empty message → updating with no messages",
			obj:  makeClusterObj(1, 1, 1, 1, -1),
			conditions: []Condition{
				NewCondition("Available", "False", "NotAvailable", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:   "updating",
			expectedTransit: true,
			expectedError:   false,
		},
		{
			name: "Deleting=True takes priority over RollingOut=True on Cluster",
			obj:  makeClusterObj(4, 5, 5, 5, 3),
			conditions: []Condition{
				NewCondition("Deleting", "True", "WaitingForWorkersDeletion", "cleanup"),
				NewCondition("RollingOut", "True", "RollingOut", "* MachineDeployment md:\n  * Rolling out 2 not up-to-date replicas"),
				NewCondition("ScalingDown", "True", "ScalingDown", "scaling down"),
			},
			expectedState:    "deleting",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"cleanup"},
		},
		{
			name: "Paused=True takes priority over RollingOut=True on Cluster",
			obj:  makeClusterObj(4, 5, 5, 5, 3),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "True", "Paused", ""),
				NewCondition("RollingOut", "True", "RollingOut", "* MachineDeployment md:\n  * Rolling out 2 not up-to-date replicas"),
			},
			expectedState:   "paused",
			expectedTransit: true,
			expectedError:   false,
		},
		{
			name: "RollingOut=False does not trigger rollingout — falls through to ScalingDown",
			obj:  makeClusterObj(3, 4, 4, 4, -1),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("RollingOut", "False", "NotRollingOut", ""),
				NewCondition("ScalingDown", "True", "ScalingDown", "scaling down"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("Available", "True", "Available", ""),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Scaling down from 4 to 3 machines"},
		},
		{
			name: "RollingOut=True on Cluster with missing upToDateReplicas — no message",
			obj:  makeClusterObj(4, 5, 5, 5, -1),
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("RollingOut", "True", "RollingOut", "* MachineDeployment md:\n  * Rolling out 2 not up-to-date replicas"),
				NewCondition("ScalingDown", "True", "ScalingDown", "scaling down"),
			},
			expectedState:    "rollingout",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: nil,
		},

		// --- Priority 7: Steady state / pass through ---
		{
			name: "Steady state — all healthy → pass through",
			obj:  makeClusterObj(2, 2, 2, 2, -1),
			conditions: []Condition{
				NewCondition("Available", "True", "Available", ""),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("WorkersAvailable", "True", "Available", ""),
				NewCondition("ControlPlaneAvailable", "True", "Available", ""),
				NewCondition("WorkerMachinesReady", "True", "Ready", ""),
				NewCondition("ControlPlaneMachinesReady", "True", "Ready", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},

		// --- Edge cases ---
		{
			name:            "No conditions → pass through",
			obj:             makeClusterObj(1, 1, 1, 1, -1),
			conditions:      []Condition{},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},
		{
			name: "Missing worker replica fields → pass through (no panic)",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "Cluster",
			},
			conditions: []Condition{
				NewCondition("Available", "True", "Available", ""),
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
			name: "ScalingUp=True but no worker replica fields → updating, no message",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "Cluster",
			},
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "False", "NotScalingDown", ""),
				NewCondition("ScalingUp", "True", "ScalingUp", "Scaling up"),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: nil,
		},
		{
			name: "ScalingDown=True but no worker replica fields → updating, no message",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "Cluster",
			},
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "True", "ScalingDown", "Scaling down"),
				NewCondition("ScalingUp", "False", "NotScalingUp", ""),
			},
			expectedState:    "updating",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: nil,
		},
		{
			name: "Deleting=True takes priority over active scale-down",
			obj:  makeClusterObj(1, 1, 1, 1, -1),
			conditions: []Condition{
				NewCondition("Deleting", "True", "WaitingForWorkersDeletion", "cleanup in progress"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("ScalingDown", "True", "ScalingDown", "Scaling down from 1 to 0"),
				NewCondition("Available", "False", "NotAvailable", "not available"),
			},
			expectedState:    "deleting",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"cleanup in progress"},
		},
		{
			name: "Paused=True takes priority over scale-up detected by replicas",
			obj:  makeClusterObj(3, 1, 1, 1, -1),
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkCAPIClusterTransitioning(tt.obj, tt.conditions, Summary{})
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

// TestCheckTransitioning_CAPIClusterDispatch verifies that checkTransitioning
// dispatches to checkCAPIClusterTransitioning for CAPI Clusters and not to
// the generic path.
func TestCheckTransitioning_CAPIClusterDispatch(t *testing.T) {
	// A CAPI Cluster with ScalingUp=True should get "updating" from the
	// CAPI Cluster path.
	capiObj := makeClusterObj(2, 1, 1, 1, -1)
	conditions := []Condition{
		NewCondition("ScalingUp", "True", "ScalingUp", "* MachineDeployment md: Scaling up from 1 to 2 replicas"),
		NewCondition("ScalingDown", "False", "NotScalingDown", ""),
		NewCondition("Deleting", "False", "NotDeleting", ""),
		NewCondition("Paused", "False", "NotPaused", ""),
		NewCondition("Available", "False", "NotAvailable", "something"),
	}
	result := checkTransitioning(capiObj, conditions, Summary{})
	assert.Equal(t, "updating", result.State)
	assert.True(t, result.Transitioning)

	// A CAPI Cluster with RollingOut=True during a rolling upgrade should
	// get state="rollingout" instead of "updating" (from ScalingDown).
	rollingClusterObj := makeClusterObj(4, 5, 5, 5, 3)
	rollingConditions := []Condition{
		NewCondition("Deleting", "False", "NotDeleting", ""),
		NewCondition("Paused", "False", "NotPaused", ""),
		NewCondition("RollingOut", "True", "RollingOut", "* MachineDeployment md:\n  * Rolling out 2 not up-to-date replicas"),
		NewCondition("ScalingDown", "True", "ScalingDown", "* MachineDeployment md: Scaling down from 4 to 3 replicas"),
		NewCondition("ScalingUp", "False", "NotScalingUp", ""),
		NewCondition("Available", "True", "Available", ""),
	}
	rollingResult := checkTransitioning(rollingClusterObj, rollingConditions, Summary{})
	assert.Equal(t, "rollingout", rollingResult.State, "rolling upgrade should produce state=rollingout")
	assert.True(t, rollingResult.Transitioning)
	assert.Equal(t, []string{"rolling out 2 not up-to-date replicas"}, rollingResult.Message)

	// A non-CAPI Cluster (e.g. management.cattle.io) should use the generic path.
	genericObj := data.Object{
		"apiVersion": "management.cattle.io/v3",
		"kind":       "Cluster",
	}
	genericConditions := []Condition{
		NewCondition("Available", "False", "MinimumReplicasUnavailable", "not available"),
	}
	genericResult := checkTransitioning(genericObj, genericConditions, Summary{})
	assert.Equal(t, "updating", genericResult.State)
	assert.True(t, genericResult.Transitioning)
}
