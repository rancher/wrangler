package summary

import (
	"testing"

	"github.com/rancher/wrangler/v3/pkg/data"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestIsOperation(t *testing.T) {
	tests := []struct {
		name     string
		obj      data.Object
		expected bool
	}{
		{
			name:     "EncryptionKeyRotation",
			obj:      data.Object{"apiVersion": "operation.cattle.io/v1alpha1", "kind": "EncryptionKeyRotation"},
			expected: true,
		},
		{
			name:     "ETCDSnapshotRestore",
			obj:      data.Object{"apiVersion": "operation.cattle.io/v1alpha1", "kind": "ETCDSnapshotRestore"},
			expected: true,
		},
		{
			name:     "ETCDSnapshotSave",
			obj:      data.Object{"apiVersion": "operation.cattle.io/v1alpha1", "kind": "ETCDSnapshotSave"},
			expected: true,
		},
		{
			name:     "CertificateRotation",
			obj:      data.Object{"apiVersion": "operation.cattle.io/v1alpha1", "kind": "CertificateRotation"},
			expected: true,
		},
		{
			name:     "unknown kind in the operation group",
			obj:      data.Object{"apiVersion": "operation.cattle.io/v1alpha1", "kind": "SomethingElse"},
			expected: false,
		},
		{
			name:     "same kind in a different group",
			obj:      data.Object{"apiVersion": "rke.cattle.io/v1", "kind": "ETCDSnapshotSave"},
			expected: false,
		},
		{
			name:     "group is only a prefix of the operation group",
			obj:      data.Object{"apiVersion": "operation.cattle.io.example/v1", "kind": "ETCDSnapshotSave"},
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
			assert.Equal(t, tt.expected, isOperation(tt.obj))
		})
	}
}

func TestCheckOperationTransitioning(t *testing.T) {
	tests := []struct {
		name             string
		conditions       []Condition
		summary          Summary
		expectedState    string
		expectedTransit  bool
		expectedError    bool
		expectedMessages []string
	}{
		{
			name: "Pending phase is transitioning, not an error",
			conditions: []Condition{
				NewCondition("Pending", "True", "WaitingForBeacon", "Waiting to acquire the beacon"),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState:    "pending",
			expectedTransit:  true,
			expectedMessages: []string{"Waiting to acquire the beacon"},
		},
		{
			name: "InProgress phase is transitioning, not an error",
			conditions: []Condition{
				NewCondition("InProgress", "True", "WaitingForPlanApplied", "Waiting in step Preflight: failing plan for machine-7qrnd"),
				NewCondition("Paused", "False", "NotPaused", ""),
				NewCondition("Pending", "False", "InProgress", "Operation now in progress"),
			},
			expectedState:    "in-progress",
			expectedTransit:  true,
			expectedMessages: []string{"Waiting in step Preflight: failing plan for machine-7qrnd"},
		},
		{
			name: "Succeeded phase is neither transitioning nor an error",
			conditions: []Condition{
				NewCondition("Pending", "False", "Finished", "Operation completed successfully"),
				NewCondition("InProgress", "False", "Finished", "Operation completed successfully"),
				NewCondition("Succeeded", "True", "Finished", "Operation completed successfully"),
				NewCondition("Failed", "False", "NotFailed", "Operation completed successfully"),
			},
			expectedState:    "succeeded",
			expectedMessages: []string{"Operation completed successfully"},
		},
		{
			name: "Failed phase is an error",
			conditions: []Condition{
				NewCondition("Pending", "False", "Finished", "Operation failed"),
				NewCondition("InProgress", "False", "Finished", "Operation failed"),
				NewCondition("Failed", "True", "PlanFailed", "restart failed for ns/secret"),
				NewCondition("Succeeded", "False", "NotSuccessful", "Operation failed"),
			},
			expectedState:    "failed",
			expectedError:    true,
			expectedMessages: []string{"restart failed for ns/secret"},
		},
		{
			name: "Canceled phase is an error and outranks a stale InProgress",
			conditions: []Condition{
				NewCondition("InProgress", "True", "WaitingForPlanApplied", "Waiting in step Preflight: stale"),
				NewCondition("Canceled", "True", "PreflightCheckFailed", "could not find server token for ns/secret"),
			},
			expectedState:    "canceled",
			expectedError:    true,
			expectedMessages: []string{"could not find server token for ns/secret"},
		},
		{
			name: "Failed outranks Canceled",
			conditions: []Condition{
				NewCondition("Canceled", "True", "PreflightCheckFailed", "canceled message"),
				NewCondition("Failed", "True", "PlanFailed", "failed message"),
			},
			expectedState:    "failed",
			expectedError:    true,
			expectedMessages: []string{"failed message"},
		},
		{
			name: "Paused is transitioning, not an error",
			conditions: []Condition{
				NewCondition("Paused", "True", "Paused", "Operation is paused"),
				NewCondition("InProgress", "True", "WaitingForPlanApplied", "Waiting in step Save"),
			},
			expectedState:    "paused",
			expectedTransit:  true,
			expectedMessages: []string{"Operation is paused"},
		},
		{
			name: "terminal conditions outrank Paused",
			conditions: []Condition{
				NewCondition("Paused", "True", "Paused", "Operation is paused"),
				NewCondition("Succeeded", "True", "Finished", "Operation completed successfully"),
			},
			expectedState:    "succeeded",
			expectedMessages: []string{"Operation completed successfully"},
		},
		{
			name: "empty condition messages are not appended",
			conditions: []Condition{
				NewCondition("Pending", "True", "WaitingForBeacon", ""),
			},
			expectedState:   "pending",
			expectedTransit: true,
		},
		{
			name:            "an unreconciled operation with no conditions is pending",
			conditions:      nil,
			expectedState:   "pending",
			expectedTransit: true,
		},
		{
			name: "no condition set to True passes through untouched",
			conditions: []Condition{
				NewCondition("Pending", "False", "InProgress", "Operation now in progress"),
				NewCondition("Paused", "False", "NotPaused", ""),
			},
			expectedState: "",
		},
		{
			name: "an error already set by checkErrors is preserved",
			conditions: []Condition{
				NewCondition("Pending", "True", "WaitingForBeacon", ""),
			},
			summary:         Summary{Error: true},
			expectedState:   "pending",
			expectedTransit: true,
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := checkOperationTransitioning(tt.conditions, tt.summary)
			assert.Equal(t, tt.expectedState, actual.State)
			assert.Equal(t, tt.expectedTransit, actual.Transitioning)
			assert.Equal(t, tt.expectedError, actual.Error)
			assert.Equal(t, tt.expectedMessages, actual.Message)
		})
	}
}

// makeOperationObj builds an operation.cattle.io object with the given phase,
// step and conditions.
func makeOperationObj(kind, phase, step string, conditions ...map[string]any) *unstructured.Unstructured {
	raw := make([]any, 0, len(conditions))
	for _, c := range conditions {
		raw = append(raw, c)
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "operation.cattle.io/v1alpha1",
		"kind":       kind,
		"metadata":   map[string]any{"name": "op", "namespace": "fleet-default"},
		"status": map[string]any{
			"phase":              phase,
			"step":               step,
			"observedGeneration": int64(1),
			"conditions":         raw,
		},
	}}
}

func cond(condType, status, reason, message string) map[string]any {
	return map[string]any{
		"type":    condType,
		"status":  status,
		"reason":  reason,
		"message": message,
	}
}

// TestSummarizeOperation exercises the whole summarizer chain, which is where
// the reported inaccuracy surfaced.
func TestSummarizeOperation(t *testing.T) {
	inProgressConditions := []map[string]any{
		cond("InProgress", "True", "WaitingForPlanApplied", "Waiting in step Preflight: failing plan for machine-7qrnd"),
		cond("Paused", "False", "NotPaused", ""),
		cond("Pending", "False", "InProgress", "Operation now in progress"),
	}

	tests := []struct {
		name     string
		obj      *unstructured.Unstructured
		expected Summary
	}{
		{
			name: "EncryptionKeyRotation in progress",
			obj:  makeOperationObj("EncryptionKeyRotation", "InProgress", "Rotate", inProgressConditions...),
			expected: Summary{
				State:         "in-progress",
				Transitioning: true,
				Message:       []string{"Waiting in step Preflight: failing plan for machine-7qrnd"},
			},
		},
		{
			name: "ETCDSnapshotSave in progress",
			obj:  makeOperationObj("ETCDSnapshotSave", "InProgress", "Save", inProgressConditions...),
			expected: Summary{
				State:         "in-progress",
				Transitioning: true,
				Message:       []string{"Waiting in step Preflight: failing plan for machine-7qrnd"},
			},
		},
		{
			name: "ETCDSnapshotRestore in progress",
			obj:  makeOperationObj("ETCDSnapshotRestore", "InProgress", "Shutdown", inProgressConditions...),
			expected: Summary{
				State:         "in-progress",
				Transitioning: true,
				Message:       []string{"Waiting in step Preflight: failing plan for machine-7qrnd"},
			},
		},
		{
			name: "freshly created, no status yet",
			obj: &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "operation.cattle.io/v1alpha1",
				"kind":       "EncryptionKeyRotation",
				"metadata":   map[string]any{"name": "op", "namespace": "fleet-default"},
			}},
			expected: Summary{State: "pending", Transitioning: true},
		},
		{
			name: "pending",
			obj: makeOperationObj("ETCDSnapshotSave", "Pending", "",
				cond("Pending", "True", "WaitingForBeacon", ""),
				cond("Paused", "False", "NotPaused", ""),
			),
			expected: Summary{State: "pending", Transitioning: true},
		},
		{
			name: "succeeded",
			obj: makeOperationObj("ETCDSnapshotSave", "Succeeded", "",
				cond("Pending", "False", "Finished", "Operation completed successfully"),
				cond("InProgress", "False", "Finished", "Operation completed successfully"),
				cond("Succeeded", "True", "Finished", "Operation completed successfully"),
				cond("Failed", "False", "NotFailed", "Operation completed successfully"),
			),
			expected: Summary{State: "succeeded", Message: []string{"Operation completed successfully"}},
		},
		{
			name: "failed",
			obj: makeOperationObj("ETCDSnapshotSave", "Failed", "Restart",
				cond("Pending", "False", "Finished", "Operation failed"),
				cond("InProgress", "False", "Finished", "Operation failed"),
				cond("Failed", "True", "PlanFailed", "restart failed for ns/secret"),
				cond("Succeeded", "False", "NotSuccessful", "Operation failed"),
			),
			expected: Summary{
				State:   "failed",
				Error:   true,
				Message: []string{"restart failed for ns/secret"},
			},
		},
		{
			name: "canceled with a stale InProgress condition",
			obj: makeOperationObj("ETCDSnapshotSave", "Canceled", "Preflight",
				cond("InProgress", "True", "WaitingForPlanApplied", "Waiting in step Preflight: stale"),
				cond("Canceled", "True", "PreflightCheckFailed", "could not find server token for ns/secret"),
			),
			expected: Summary{
				State:   "canceled",
				Error:   true,
				Message: []string{"could not find server token for ns/secret"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Summarize(tt.obj))
		})
	}
}
