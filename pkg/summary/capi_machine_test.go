package summary

import (
	"testing"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/stretchr/testify/assert"
)

func TestIsCAPIMachine(t *testing.T) {
	tests := []struct {
		name     string
		obj      data.Object
		expected bool
	}{
		{
			name: "CAPI v1beta2 Machine",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "Machine",
			},
			expected: true,
		},
		{
			name: "CAPI v1beta1 Machine",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta1",
				"kind":       "Machine",
			},
			expected: true,
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
			name: "CAPI v1beta2 MachineSet",
			obj: data.Object{
				"apiVersion": "cluster.x-k8s.io/v1beta2",
				"kind":       "MachineSet",
			},
			expected: false,
		},
		{
			name: "non-CAPI Machine kind",
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
			assert.Equal(t, tt.expected, isCAPIMachine(tt.obj))
		})
	}
}

func TestParseReadyFirstBullet(t *testing.T) {
	tests := []struct {
		name           string
		message        string
		expectedDetail string
		expectedPrefix string
	}{
		{
			name:           "empty message",
			message:        "",
			expectedDetail: "",
			expectedPrefix: "",
		},
		{
			name:           "single bullet BootstrapConfigReady",
			message:        "* BootstrapConfigReady: RKEBootstrap status.initialization.dataSecretCreated is false",
			expectedDetail: "RKEBootstrap status.initialization.dataSecretCreated is false",
			expectedPrefix: "BootstrapConfigReady",
		},
		{
			name:           "single bullet InfrastructureReady",
			message:        "* InfrastructureReady: creating server of kind (DigitaloceanMachine)",
			expectedDetail: "creating server of kind (DigitaloceanMachine)",
			expectedPrefix: "InfrastructureReady",
		},
		{
			name:           "single bullet NodeHealthy",
			message:        "* NodeHealthy: Waiting for Cluster control plane to be initialized",
			expectedDetail: "Waiting for Cluster control plane to be initialized",
			expectedPrefix: "NodeHealthy",
		},
		{
			name:           "multiple bullets - takes first only",
			message:        "* BootstrapConfigReady: bootstrap not ready\n* InfrastructureReady: infra not ready\n* NodeHealthy: waiting",
			expectedDetail: "bootstrap not ready",
			expectedPrefix: "BootstrapConfigReady",
		},
		{
			name:           "multiple bullets starting with InfrastructureReady",
			message:        "* InfrastructureReady: creating server...\n* NodeHealthy: Waiting for CP init",
			expectedDetail: "creating server...",
			expectedPrefix: "InfrastructureReady",
		},
		{
			name:           "no bullet prefix",
			message:        "some plain message",
			expectedDetail: "some plain message",
			expectedPrefix: "",
		},
		{
			name:           "bullet without colon separator",
			message:        "* SomeCondition without colon",
			expectedDetail: "SomeCondition without colon",
			expectedPrefix: "",
		},
		{
			name:           "nested sub-bullet: NodeHealthy with empty inline detail",
			message:        "* NodeHealthy:\n  * Node.AllConditions: Kubelet stopped posting node status.",
			expectedDetail: "Kubelet stopped posting node status.",
			expectedPrefix: "NodeHealthy",
		},
		{
			name:           "nested sub-bullet: multiple sub-bullets takes first",
			message:        "* NodeHealthy:\n  * Node.AllConditions: first issue\n  * Node.OtherCondition: second issue",
			expectedDetail: "first issue",
			expectedPrefix: "NodeHealthy",
		},
		{
			name:           "nested sub-bullet: sub-bullet without colon separator",
			message:        "* NodeHealthy:\n  * some plain sub-bullet",
			expectedDetail: "some plain sub-bullet",
			expectedPrefix: "NodeHealthy",
		},
		{
			name:           "nested sub-bullet: InfrastructureReady with empty inline detail",
			message:        "* InfrastructureReady:\n  * SubCondition: infra detail here",
			expectedDetail: "infra detail here",
			expectedPrefix: "InfrastructureReady",
		},
		{
			name:           "nested sub-bullet: no sub-bullets found (only non-bullet lines)",
			message:        "* NodeHealthy:\n  some non-bullet line",
			expectedDetail: "",
			expectedPrefix: "NodeHealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail, prefix := parseReadyFirstBullet(tt.message)
			assert.Equal(t, tt.expectedDetail, detail)
			assert.Equal(t, tt.expectedPrefix, prefix)
		})
	}
}

func TestCheckCAPIMachineTransitioning(t *testing.T) {
	tests := []struct {
		name             string
		conditions       []Condition
		expectedState    string
		expectedTransit  bool
		expectedError    bool
		expectedMessages []string
	}{
		// --- Priority 1: Deleting ---
		{
			name: "Deleting=True takes absolute priority",
			conditions: []Condition{
				NewCondition("Reconciled", "Unknown", "Waiting", "reconciling something"),
				NewCondition("Deleting", "True", "Deleting", "machine is being deleted"),
				NewCondition("Ready", "False", "NotReady", "* InfrastructureReady: infra not ready"),
			},
			expectedState:    "deleting",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"machine is being deleted"},
		},
		{
			name: "Deleting=True with empty message",
			conditions: []Condition{
				NewCondition("Deleting", "True", "Deleting", ""),
			},
			expectedState:    "deleting",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: nil,
		},
		{
			name: "Deleting=True with drain message rewrite",
			conditions: []Condition{
				NewCondition("Deleting", "True", "Draining", "Drain not completed yet (1/3 nodes drained)"),
			},
			expectedState:    "deleting",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Draining node"},
		},

		// --- Priority 2: Paused ---
		{
			name: "Paused=True takes priority over Reconciled and Ready",
			conditions: []Condition{
				NewCondition("Reconciled", "Unknown", "Waiting", "reconciling"),
				NewCondition("Paused", "True", "Paused", "machine is paused"),
				NewCondition("Ready", "False", "NotReady", "* InfrastructureReady: infra not ready"),
			},
			expectedState:   "paused",
			expectedTransit: true,
			expectedError:   false,
		},
		{
			name: "Paused=False is ignored (normal state)",
			conditions: []Condition{
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Ready", "True", "Ready", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},

		// --- Priority 3: Reconciled not True ---
		{
			name: "Reconciled=Unknown → reconciling with message",
			conditions: []Condition{
				NewCondition("Ready", "Unknown", "ReadyUnknown", "* NodeHealthy: waiting"),
				NewCondition("Reconciled", "Unknown", "Waiting", "waiting for agent to check in and apply initial plan"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:    "reconciling",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"waiting for agent to check in and apply initial plan"},
		},
		{
			name: "Reconciled=Unknown with empty message",
			conditions: []Condition{
				NewCondition("Ready", "Unknown", "ReadyUnknown", "* NodeHealthy: waiting"),
				NewCondition("Reconciled", "Unknown", "Waiting", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:   "reconciling",
			expectedTransit: true,
			expectedError:   false,
		},
		{
			name: "Reconciled=False → reconciling with error",
			conditions: []Condition{
				NewCondition("Ready", "True", "Ready", ""),
				NewCondition("Reconciled", "False", "ReconcileError", "failed to reconcile machine"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:    "reconciling",
			expectedTransit:  false,
			expectedError:    true,
			expectedMessages: []string{"failed to reconcile machine"},
		},
		{
			name: "Reconciled=False with empty message → reconciling with error, no messages",
			conditions: []Condition{
				NewCondition("Ready", "True", "Ready", ""),
				NewCondition("Reconciled", "False", "ReconcileError", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:    "reconciling",
			expectedTransit:  false,
			expectedError:    true,
			expectedMessages: nil,
		},
		{
			name: "Reconciled=Unknown takes priority over Ready=False",
			conditions: []Condition{
				NewCondition("Ready", "False", "NotReady", "* InfrastructureReady: infra not ready"),
				NewCondition("Reconciled", "Unknown", "Waiting", "reconciling"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:    "reconciling",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"reconciling"},
		},

		// --- Priority 4: Ready=False ---
		{
			name: "Ready=False, first bullet BootstrapConfigReady → waitingforinfrastructure",
			conditions: []Condition{
				NewCondition("Ready", "False", "NotReady",
					"* BootstrapConfigReady: RKEBootstrap status.initialization.dataSecretCreated is false\n"+
						"* InfrastructureReady: DigitaloceanMachine status.initialization.provisioned is false\n"+
						"* NodeHealthy: Waiting for Cluster control plane to be initialized"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:    "waitingforinfrastructure",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: nil,
		},
		{
			name: "Ready=False, first bullet InfrastructureReady → waitingfornoderef",
			conditions: []Condition{
				NewCondition("Ready", "False", "NotReady",
					"* InfrastructureReady: creating server [fleet-default/prod-jiaqi-pa-qxwqc-9gdk2] of kind (DigitaloceanMachine) for machine prod-jiaqi-pa-qxwqc-9gdk2 in infrastructure provider\n"+
						"* NodeHealthy: Waiting for Cluster control plane to be initialized"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:    "waitingfornoderef",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"creating server [fleet-default/prod-jiaqi-pa-qxwqc-9gdk2] of kind (DigitaloceanMachine) for machine prod-jiaqi-pa-qxwqc-9gdk2 in infrastructure provider"},
		},
		{
			name: "Ready=False, unknown first bullet → pass through (no state set)",
			conditions: []Condition{
				NewCondition("Ready", "False", "NotReady", "* SomeOtherCondition: something happened"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},
		{
			name: "Ready=False, InfrastructureReady detail suppressed when ending with 'status.initialization.provisioned is false'",
			conditions: []Condition{
				NewCondition("Ready", "False", "NotReady",
					"* InfrastructureReady: DigitaloceanMachine status.initialization.provisioned is false\n"+
						"* NodeHealthy: Waiting for Cluster control plane to be initialized"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:   "waitingfornoderef",
			expectedTransit: true,
			expectedError:   false,
			// Detail is suppressed — no messages expected.
		},
		{
			name: "Ready=False with empty message → pass through",
			conditions: []Condition{
				NewCondition("Ready", "False", "NotReady", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},

		// --- Priority 5: Ready=Unknown ---
		{
			name: "Ready=Unknown with NodeHealthy bullet → reconciling",
			conditions: []Condition{
				NewCondition("Ready", "Unknown", "ReadyUnknown", "* NodeHealthy: Waiting for Cluster control plane to be initialized"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:    "reconciling",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Waiting for Cluster control plane to be initialized"},
		},
		{
			name: "Ready=Unknown, Reconciled=True → reconciling (rule 5, not rule 3)",
			conditions: []Condition{
				NewCondition("Ready", "Unknown", "ReadyUnknown", "* NodeHealthy: Waiting for Cluster control plane to be initialized"),
				NewCondition("Reconciled", "True", "Reconciled", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:    "reconciling",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Waiting for Cluster control plane to be initialized"},
		},
		{
			name: "Ready=Unknown with multiple bullets → reconciling with first bullet detail",
			conditions: []Condition{
				NewCondition("Ready", "Unknown", "ReadyUnknown", "* InfrastructureReady: some issue\n* NodeHealthy: waiting"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:    "reconciling",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"some issue"},
		},
		{
			name: "Ready=Unknown with nested sub-bullet (NodeHealthy empty detail) → reconciling with sub-bullet detail",
			conditions: []Condition{
				NewCondition("Ready", "Unknown", "ReadyUnknown", "* NodeHealthy:\n  * Node.AllConditions: Kubelet stopped posting node status."),
				NewCondition("Reconciled", "True", "Reconciled", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:    "reconciling",
			expectedTransit:  true,
			expectedError:    false,
			expectedMessages: []string{"Kubelet stopped posting node status."},
		},
		{
			name: "Ready=Unknown with NodeHealthy detail suppressed when ending with 'to report spec.providerID'",
			conditions: []Condition{
				NewCondition("Ready", "Unknown", "ReadyUnknown", "* NodeHealthy: Waiting for Node controller to report spec.providerID"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:   "reconciling",
			expectedTransit: true,
			expectedError:   false,
			// Detail is suppressed because it ends with "to report spec.providerID".
		},
		{
			name: "Ready=Unknown with empty message → reconciling with no messages",
			conditions: []Condition{
				NewCondition("Ready", "Unknown", "ReadyUnknown", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:   "reconciling",
			expectedTransit: true,
			expectedError:   false,
		},

		// --- Priority 6: Ready=True ---
		{
			name: "Ready=True → pass through (no state set)",
			conditions: []Condition{
				NewCondition("Ready", "True", "Ready", ""),
				NewCondition("Reconciled", "True", "Reconciled", ""),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},

		// --- Edge cases ---
		{
			name:            "no conditions → pass through",
			conditions:      []Condition{},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},
		{
			name: "no Ready or Reconciled conditions → pass through",
			conditions: []Condition{
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Deleting", "False", "NotDeleting", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},
		{
			name: "Reconciled=True is skipped (not a problem state)",
			conditions: []Condition{
				NewCondition("Ready", "True", "Ready", ""),
				NewCondition("Reconciled", "True", "Reconciled", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},
		{
			name: "Deleting=False does not trigger removing",
			conditions: []Condition{
				NewCondition("Deleting", "False", "NotDeleting", ""),
				NewCondition("Ready", "True", "Ready", ""),
				NewCondition("Reconciled", "True", "Reconciled", ""),
			},
			expectedState:   "",
			expectedTransit: false,
			expectedError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkCAPIMachineTransitioning(tt.conditions, Summary{})
			assert.Equal(t, tt.expectedState, result.State)
			assert.Equal(t, tt.expectedTransit, result.Transitioning)
			assert.Equal(t, tt.expectedError, result.Error)
			if tt.expectedMessages != nil {
				assert.Equal(t, tt.expectedMessages, result.Message)
			} else {
				assert.Empty(t, result.Message)
			}
		})
	}
}

// TestCheckTransitioning_CAPIMachineDispatch verifies that checkTransitioning
// dispatches to checkCAPIMachineTransitioning for CAPI Machines and to
// checkGenericTransitioning for everything else.
func TestCheckTransitioning_CAPIMachineDispatch(t *testing.T) {
	// A CAPI Machine with Reconciled=Unknown should get "reconciling" from
	// the CAPI path, not the generic TransitioningUnknown path.
	capiObj := data.Object{
		"apiVersion": "cluster.x-k8s.io/v1beta2",
		"kind":       "Machine",
	}
	conditions := []Condition{
		NewCondition("Reconciled", "Unknown", "Waiting", "waiting for agent"),
		NewCondition("Ready", "Unknown", "ReadyUnknown", "* NodeHealthy: waiting"),
		NewCondition("Paused", "False", "NotPaused", ""),
		NewCondition("Deleting", "False", "NotDeleting", ""),
	}
	result := checkTransitioning(capiObj, conditions, Summary{})
	assert.Equal(t, "reconciling", result.State)
	assert.True(t, result.Transitioning)

	// A non-CAPI object with Reconciled=Unknown should use the generic path.
	// In TransitioningUnknown, Reconciled maps to "reconciling" for Unknown status.
	genericObj := data.Object{
		"apiVersion": "management.cattle.io/v3",
		"kind":       "Cluster",
	}
	genericConditions := []Condition{
		NewCondition("Reconciled", "Unknown", "Waiting", "waiting for something"),
	}
	genericResult := checkTransitioning(genericObj, genericConditions, Summary{})
	assert.Equal(t, "reconciling", genericResult.State)
	assert.True(t, genericResult.Transitioning)
}

// TestCheckTransitioning_NonCAPIMachineUnchanged verifies that the generic
// path remains unchanged for non-CAPI objects.
func TestCheckTransitioning_NonCAPIMachineUnchanged(t *testing.T) {
	obj := data.Object{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
	}
	// Available=False in TransitioningFalse maps to "updating"
	conditions := []Condition{
		NewCondition("Available", "False", "MinimumReplicasUnavailable", "Deployment does not have minimum availability"),
	}
	result := checkTransitioning(obj, conditions, Summary{})
	assert.Equal(t, "updating", result.State)
	assert.True(t, result.Transitioning)
	assert.False(t, result.Error)
}
