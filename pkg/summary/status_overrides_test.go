package summary

import (
	"testing"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/stretchr/testify/assert"
)

func TestFindOverride(t *testing.T) {
	overrides := []StatusOverride{
		{
			ConditionType: "Updating",
			Kind:          "Machine",
			APIGroup:      "cluster.x-k8s.io",
			Action:        OverrideSkip,
		},
		{
			ConditionType:   "MachinesReady",
			Status:          "Unknown",
			MessageContains: "NodeHealthy: Waiting for",
			Action:          OverrideTransitioning,
		},
		{
			ConditionType: "NodeHealthy",
			Reason:        "InspectionFailed",
			Action:        OverrideTransitioning,
			StateOverride: "provisioning",
		},
		{
			ConditionType:   "Updated",
			Reason:          "Waiting",
			Status:          "Unknown",
			MessageContains: "CAPI cluster or RKEControlPlane is paused",
			Kind:            "Cluster",
			APIVersion:      "management.cattle.io/v3",
			Action:          OverrideTransitioning,
			StateOverride:   "paused",
		},
		{
			ConditionType:   "Updated",
			Reason:          "Waiting",
			Status:          "Unknown",
			MessageContains: "CAPI cluster or RKEControlPlane is paused",
			Kind:            "Cluster",
			APIGroup:        "provisioning.cattle.io",
			Action:          OverrideTransitioning,
			StateOverride:   "paused",
		},
	}

	tests := []struct {
		name     string
		obj      data.Object
		cond     Condition
		expected *StatusOverride
	}{
		{
			name: "Updating on CAPI Machine matches skip override",
			obj: data.Object{
				"kind":       "Machine",
				"apiVersion": "cluster.x-k8s.io/v1beta1",
			},
			cond:     NewCondition("Updating", "True", "", ""),
			expected: &overrides[0],
		},
		{
			name: "Updating on CAPI Machine v1beta2 matches skip override (APIGroup prefix)",
			obj: data.Object{
				"kind":       "Machine",
				"apiVersion": "cluster.x-k8s.io/v1beta2",
			},
			cond:     NewCondition("Updating", "True", "", ""),
			expected: &overrides[0],
		},
		{
			name: "Updating on non-Machine does not match",
			obj: data.Object{
				"kind":       "Deployment",
				"apiVersion": "apps/v1",
			},
			cond:     NewCondition("Updating", "True", "", ""),
			expected: nil,
		},
		{
			name: "MachinesReady Unknown with matching message matches transitioning override",
			obj:  data.Object{},
			cond: NewCondition("MachinesReady", "Unknown", "",
				"Condition NodeHealthy: Waiting for node to become healthy"),
			expected: &overrides[1],
		},
		{
			name:     "MachinesReady Unknown without matching message does not match",
			obj:      data.Object{},
			cond:     NewCondition("MachinesReady", "Unknown", "", "some other message"),
			expected: nil,
		},
		{
			name:     "MachinesReady False does not match (status mismatch)",
			obj:      data.Object{},
			cond:     NewCondition("MachinesReady", "False", "", "NodeHealthy: Waiting for"),
			expected: nil,
		},
		{
			name:     "NodeHealthy with InspectionFailed matches transitioning override with state override",
			obj:      data.Object{},
			cond:     NewCondition("NodeHealthy", "False", "InspectionFailed", "inspection failed"),
			expected: &overrides[2],
		},
		{
			name:     "NodeHealthy with other reason does not match",
			obj:      data.Object{},
			cond:     NewCondition("NodeHealthy", "False", "SomeOtherReason", ""),
			expected: nil,
		},
		{
			name:     "Unrelated condition does not match any override",
			obj:      data.Object{},
			cond:     NewCondition("Ready", "True", "", ""),
			expected: nil,
		},
		{
			name: "Updated on management.cattle.io/v3 Cluster with paused message matches exact APIVersion",
			obj: data.Object{
				"kind":       "Cluster",
				"apiVersion": "management.cattle.io/v3",
			},
			cond:     NewCondition("Updated", "Unknown", "Waiting", "CAPI cluster or RKEControlPlane is paused"),
			expected: &overrides[3],
		},
		{
			name: "Updated on provisioning.cattle.io Cluster with paused message matches APIGroup",
			obj: data.Object{
				"kind":       "Cluster",
				"apiVersion": "provisioning.cattle.io/v1",
			},
			cond:     NewCondition("Updated", "Unknown", "Waiting", "CAPI cluster or RKEControlPlane is paused"),
			expected: &overrides[4],
		},
		{
			name: "Updated on management.cattle.io/v3 Cluster without paused message does not match",
			obj: data.Object{
				"kind":       "Cluster",
				"apiVersion": "management.cattle.io/v3",
			},
			cond:     NewCondition("Updated", "Unknown", "Waiting", "some other message"),
			expected: nil,
		},
		{
			name: "Updated on wrong kind does not match",
			obj: data.Object{
				"kind":       "MachineDeployment",
				"apiVersion": "management.cattle.io/v3",
			},
			cond:     NewCondition("Updated", "Unknown", "Waiting", "CAPI cluster or RKEControlPlane is paused"),
			expected: nil,
		},
		{
			name: "Updated with wrong reason does not match",
			obj: data.Object{
				"kind":       "Cluster",
				"apiVersion": "management.cattle.io/v3",
			},
			cond:     NewCondition("Updated", "Unknown", "OtherReason", "CAPI cluster or RKEControlPlane is paused"),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findOverride(overrides, tt.obj, tt.cond)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestApplyOverride(t *testing.T) {
	tests := []struct {
		name            string
		override        StatusOverride
		cond            Condition
		initialState    string
		newState        string
		expectedSkip    bool
		expectedSummary Summary
	}{
		{
			name:            "OverrideSkip returns true",
			override:        StatusOverride{Action: OverrideSkip},
			cond:            NewCondition("Updating", "True", "", ""),
			newState:        "updating",
			expectedSkip:    true,
			expectedSummary: Summary{},
		},
		{
			name:         "OverrideTransitioning sets transitioning",
			override:     StatusOverride{Action: OverrideTransitioning},
			cond:         NewCondition("MachinesReady", "Unknown", "", "NodeHealthy: Waiting for node"),
			newState:     "updating",
			expectedSkip: true,
			expectedSummary: Summary{
				Transitioning: true,
				State:         "updating",
				Message:       []string{"NodeHealthy: Waiting for node"},
			},
		},
		{
			name:         "OverrideTransitioning with reason placeholder",
			override:     StatusOverride{Action: OverrideTransitioning},
			cond:         NewCondition("ScalingUp", "True", "ScalingUpReason", "scaling up msg"),
			newState:     "ScalingUp/ScalingUpReason",
			expectedSkip: true,
			expectedSummary: Summary{
				Transitioning: true,
				State:         "ScalingUp/ScalingUpReason",
				Message:       []string{"scaling up msg"},
			},
		},
		{
			name:         "OverrideError sets error",
			override:     StatusOverride{Action: OverrideError},
			cond:         NewCondition("SomeCond", "Unknown", "", "bad thing happened"),
			newState:     "some-state",
			expectedSkip: true,
			expectedSummary: Summary{
				Error:   true,
				State:   "some-state",
				Message: []string{"bad thing happened"},
			},
		},
		{
			name:         "OverrideTransitioning with StateOverride replaces state name",
			override:     StatusOverride{Action: OverrideTransitioning, StateOverride: "provisioning"},
			cond:         NewCondition("NodeHealthy", "False", "InspectionFailed", "inspection failed"),
			newState:     "NodeHealthy",
			expectedSkip: true,
			expectedSummary: Summary{
				Transitioning: true,
				State:         "provisioning",
				Message:       []string{"inspection failed"},
			},
		},
		{
			name:         "OverrideError with StateOverride replaces state name",
			override:     StatusOverride{Action: OverrideError, StateOverride: "custom-error"},
			cond:         NewCondition("SomeCond", "Unknown", "", "something broke"),
			newState:     "some-state",
			expectedSkip: true,
			expectedSummary: Summary{
				Error:   true,
				State:   "custom-error",
				Message: []string{"something broke"},
			},
		},
		{
			name:         "OverrideTransitioning with empty message does not append empty string",
			override:     StatusOverride{Action: OverrideTransitioning, StateOverride: "provisioning"},
			cond:         NewCondition("NodeHealthy", "False", "InspectionFailed", ""),
			newState:     "NodeHealthy",
			expectedSkip: true,
			expectedSummary: Summary{
				Transitioning: true,
				State:         "provisioning",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := Summary{State: tt.initialState}
			newState := tt.newState
			skip := applyOverride(&tt.override, tt.cond, &summary, newState)
			assert.Equal(t, tt.expectedSkip, skip)
			assert.Equal(t, tt.expectedSummary, summary)
		})
	}
}

func TestCheckTransitioning_OverrideSkip_CAPIMachine(t *testing.T) {
	obj := data.Object{
		"kind":       "Machine",
		"apiVersion": "cluster.x-k8s.io/v1beta1",
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"type":    "Updating",
					"status":  "True",
					"reason":  "",
					"message": "in-place update in progress",
				},
			},
		},
	}
	conditions := []Condition{
		NewCondition("Updating", "True", "", "in-place update in progress"),
	}
	summary := checkTransitioning(obj, conditions, Summary{})
	// "Updating" should be skipped for CAPI Machine, summary unchanged
	assert.Equal(t, "", summary.State)
	assert.False(t, summary.Transitioning)
	assert.False(t, summary.Error)
}

func TestCheckTransitioning_OverrideSkip_NonCAPIMachine(t *testing.T) {
	obj := data.Object{
		"kind":       "SomeResource",
		"apiVersion": "cattle.io/v1",
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"type":    "Updating",
					"status":  "False",
					"reason":  "",
					"message": "update failed",
				},
			},
		},
	}
	conditions := []Condition{
		NewCondition("Updating", "False", "", "update failed"),
	}
	summary := checkTransitioning(obj, conditions, Summary{})
	// "Updating" should NOT be skipped for non-CAPI object, False => error
	assert.Equal(t, "updating", summary.State)
	assert.True(t, summary.Error)
}

func TestCheckTransitioning_MachinesReadyUnknown_WithNodeHealthyWaiting(t *testing.T) {
	obj := data.Object{
		"kind":       "MachineDeployment",
		"apiVersion": "cluster.x-k8s.io/v1beta1",
	}
	conditions := []Condition{
		NewCondition("MachinesReady", "Unknown", "", "Condition NodeHealthy: Waiting for node to become healthy"),
	}
	summary := checkTransitioning(obj, conditions, Summary{})
	// Should be transitioning, NOT error
	assert.True(t, summary.Transitioning)
	assert.False(t, summary.Error)
	assert.Equal(t, "updating", summary.State)
}

func TestCheckTransitioning_MachinesReadyUnknown_WithoutNodeHealthyWaiting(t *testing.T) {
	obj := data.Object{
		"kind":       "MachineDeployment",
		"apiVersion": "cluster.x-k8s.io/v1beta1",
	}
	conditions := []Condition{
		NewCondition("MachinesReady", "Unknown", "", "some other message"),
	}
	summary := checkTransitioning(obj, conditions, Summary{})
	// Default behavior: Unknown in TransitioningFalse => error
	assert.True(t, summary.Error)
	assert.False(t, summary.Transitioning)
	assert.Equal(t, "updating", summary.State)
}

func TestCheckTransitioning_NodeHealthy_InspectionFailed(t *testing.T) {
	obj := data.Object{
		"kind":       "Machine",
		"apiVersion": "cluster.x-k8s.io/v1beta1",
	}
	conditions := []Condition{
		NewCondition("NodeHealthy", "False", "InspectionFailed", "inspection failed"),
	}
	summary := checkTransitioning(obj, conditions, Summary{})
	// InspectionFailed should override state to "provisioning" and mark as transitioning
	assert.Equal(t, "provisioning", summary.State)
	assert.True(t, summary.Transitioning)
	assert.False(t, summary.Error)
	assert.Equal(t, []string{"inspection failed"}, summary.Message)
}

func TestCheckTransitioning_AvailableSkipped_CAPIMachine(t *testing.T) {
	obj := data.Object{
		"kind":       "Machine",
		"apiVersion": "cluster.x-k8s.io/v1beta1",
	}
	conditions := []Condition{
		NewCondition("Available", "False", "", "not available yet"),
	}
	summary := checkTransitioning(obj, conditions, Summary{})
	// "Available" should be skipped for CAPI Machine
	assert.Equal(t, "", summary.State)
	assert.False(t, summary.Transitioning)
	assert.False(t, summary.Error)
}

func TestCheckTransitioning_MultiLineMessage_Preserved(t *testing.T) {
	multiLineMsg := "* Machine docluster-jiaqi-pa-lfp7l-lzl4r:\n" +
		"  * InfrastructureReady: creating server\n" +
		"  * NodeHealthy: Waiting for DigitaloceanMachine to report spec.providerID"

	obj := data.Object{
		"kind":       "MachineDeployment",
		"apiVersion": "cluster.x-k8s.io/v1beta1",
	}
	conditions := []Condition{
		NewCondition("MachinesReady", "False", "NotReady", multiLineMsg),
	}
	summary := checkTransitioning(obj, conditions, Summary{})

	// The multi-line message must be preserved as a single string with embedded newlines
	assert.Len(t, summary.Message, 1)
	assert.Contains(t, summary.Message[0], "\n")
	assert.Equal(t, multiLineMsg, summary.Message[0])
}

func TestCheckTransitioning_ManagementCluster_Paused(t *testing.T) {
	obj := data.Object{
		"kind":       "Cluster",
		"apiVersion": "management.cattle.io/v3",
	}
	conditions := []Condition{
		NewCondition("Updated", "Unknown", "Waiting", "CAPI cluster or RKEControlPlane is paused"),
	}
	summary := checkTransitioning(obj, conditions, Summary{})
	// Paused cluster should have state "paused" and be transitioning, not error
	assert.Equal(t, "paused", summary.State)
	assert.True(t, summary.Transitioning)
	assert.False(t, summary.Error)
}

func TestCheckTransitioning_ProvisioningCluster_Paused_Updated(t *testing.T) {
	obj := data.Object{
		"kind":       "Cluster",
		"apiVersion": "provisioning.cattle.io/v1",
	}
	conditions := []Condition{
		NewCondition("Updated", "Unknown", "Waiting", "CAPI cluster or RKEControlPlane is paused"),
	}
	summary := checkTransitioning(obj, conditions, Summary{})
	// Paused provisioning cluster should have state "paused" and be transitioning
	assert.Equal(t, "paused", summary.State)
	assert.True(t, summary.Transitioning)
	assert.False(t, summary.Error)
}

func TestCheckTransitioning_ProvisioningCluster_Paused_Provisioned(t *testing.T) {
	obj := data.Object{
		"kind":       "Cluster",
		"apiVersion": "provisioning.cattle.io/v1",
	}
	conditions := []Condition{
		NewCondition("Provisioned", "Unknown", "Waiting", "CAPI cluster or RKEControlPlane is paused"),
	}
	summary := checkTransitioning(obj, conditions, Summary{})
	// Paused provisioning cluster via Provisioned condition should have state "paused"
	assert.Equal(t, "paused", summary.State)
	assert.True(t, summary.Transitioning)
	assert.False(t, summary.Error)
}

func TestCheckTransitioning_ManagementCluster_NotPaused(t *testing.T) {
	obj := data.Object{
		"kind":       "Cluster",
		"apiVersion": "management.cattle.io/v3",
	}
	conditions := []Condition{
		NewCondition("Updated", "Unknown", "Waiting", "some other message"),
	}
	summary := checkTransitioning(obj, conditions, Summary{})
	// Non-paused cluster should follow default behavior (Unknown in TransitioningUnknown = transitioning)
	assert.Equal(t, "updating", summary.State)
	assert.True(t, summary.Transitioning)
	assert.False(t, summary.Error)
}
